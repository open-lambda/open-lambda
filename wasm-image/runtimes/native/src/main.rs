#![feature(impl_trait_in_assoc_type)]

use std::env::args;
use std::fs::File;
use std::io::Write;
use std::os::unix::io::FromRawFd;
use std::os::unix::net::UnixListener as StdUnixListener;
use std::os::unix::process::ExitStatusExt;
use std::process::Stdio;

use hyper::body::{Bytes, Incoming};
use hyper::server::conn::http1;
use hyper::{Method, Request, Response, StatusCode};

use http_body_util::{BodyExt, Full};

use tokio::io::AsyncReadExt;
use tokio::net::UnixListener;
use tokio::process::Command;
use tokio::select;
use tokio::signal::unix::{SignalKind, signal};

use nix::sched::{CloneFlags, unshare};
use nix::unistd::{ForkResult, fork, getpid};

use hyper_util::rt::tokio::TokioIo;

struct Service {}

impl hyper::service::Service<Request<Incoming>> for Service {
    type Response = Response<Full<Bytes>>;
    type Error = hyper::Error;
    type Future = impl std::future::Future<Output = hyper::Result<Response<Full<Bytes>>>>;

    fn call(&self, req: Request<Incoming>) -> Self::Future {
        Self::handle_request(req)
    }
}

impl Service {
    async fn handle_request(req: Request<Incoming>) -> hyper::Result<hyper::Response<Full<Bytes>>> {
        log::trace!("Got new request: {req:?}");

        let method = req.method().clone();
        let args = req.collect().await?.to_bytes().to_vec();

        if method == Method::POST {
            Self::execute_function(args).await
        } else {
            panic!("Got unexpected request");
        }
    }

    async fn execute_function(args: Vec<u8>) -> hyper::Result<Response<Full<Bytes>>> {
        use std::io::Read;

        let arg_str = String::from_utf8(args).unwrap();
        log::debug!("Executing function with arg `{arg_str}`");

        let status_code;

        let mut child = Command::new("/handler/f.bin")
            .arg(arg_str)
            .env("RUST_LOG", "info")
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .expect("Failed to spawn lambda process");

        let child_future = child.wait();

        let mut sighandler = signal(SignalKind::terminate()).expect("Failed to install sighandler");
        let sig_future = sighandler.recv();

        log::debug!("Waiting for process to terminate or signal");

        let error = select! {
            _ = sig_future => {
                log::info!("Got ctrl+c");
                child.kill().await.expect("Failed to kill child");
                child.wait().await.unwrap();

                std::process::exit(0);
            }
            res = child_future => {
                let status = res.unwrap();

                if status.success() {
                    None 
                } else if let Some(code) = status.code() {
                    Some(format!("Function failed with error_code: {code}"))
                } else if let Some(signal) = status.signal() {
                    Some(format!("Function failed with signal: {signal}"))
                } else {
                    panic!("Function failed for unknown reason");
                }
            }
        };

        let body = if let Some(err_str) = error {
            status_code = StatusCode::INTERNAL_SERVER_ERROR;
            log::error!("{err_str}");
            err_str
        } else { 
            log::debug!("Function returned successfully");

            match File::open("/tmp/output") {
                Ok(mut f) => {
                    let mut jstr = String::new();
                    f.read_to_string(&mut jstr).unwrap();

                    log::debug!("Got response: {jstr}");

                    status_code = StatusCode::OK;
                    jstr
                }
                Err(err) => {
                    if err.kind() != std::io::ErrorKind::NotFound {
                        let err_str = format!("Got unexpected error: {err:?}");
                        log::error!("{err_str}");

                        status_code = StatusCode::INTERNAL_SERVER_ERROR;
                        err_str
                    } else {
                        log::debug!("Function did not give a response");

                        status_code = StatusCode::OK;
                        String::default()
                    }
                }
            }
        };

        let mut stdout = String::default();
        let mut stderr = String::default();

        child
            .stdout
            .take()
            .expect("Failed to get child stdout")
            .read_to_string(&mut stdout)
            .await
            .unwrap();
        child
            .stderr
            .take()
            .expect("Failed to get child stderr")
            .read_to_string(&mut stderr)
            .await
            .unwrap();

        if stdout.is_empty() {
            log::debug!("Program has no stdout output");
        } else {
            let mut log_line = String::from("Process stdout:\n");

            for line in stdout.split('\n') {
                log_line += format!("    {line}\n").as_str();
            }

            log::trace!("{log_line}");
        }
        if stderr.is_empty() {
            log::debug!("Program has no stderr output");
        } else {
            let mut log_line = String::from("Process stderr:\n");

            for line in stderr.split('\n') {
                log_line += format!("    {}\n", line).as_str();
            }

            log::error!("{log_line}");
        }

        let response = Response::builder()
            .status(status_code)
            .body(body.into())
            .unwrap();

        Ok(response)
    }
}

fn main() {
    if let Err(err) = simple_logging::log_to_file("/host/ol-runtime.log", log::LevelFilter::Info) {
        println!("Failed to create logfile: {err}");
    }

    let socket_path = "/host/ol.sock";
    let listener = StdUnixListener::bind(socket_path).expect("Failed to bind UNIX socket");

    let mut argv = args();
    argv.next().unwrap();

    let pid = getpid();

    let cgroup_count: i32 = if let Some(arg) = argv.next() {
        arg.parse().unwrap()
    } else {
        0
    };

    for i in 0..cgroup_count {
        let fd = 3 + i;
        let mut f = unsafe { File::from_raw_fd(fd) };

        f.write_all(format!("{pid}").as_bytes()).unwrap();
        log::trace!("Joined cgroup, closing FD {fd}'");
    }

    unshare(CloneFlags::CLONE_NEWUTS | CloneFlags::CLONE_NEWPID | CloneFlags::CLONE_NEWIPC)
        .unwrap();

    if let ForkResult::Parent { .. } = unsafe { fork().expect("Fork failed") } {
        std::process::exit(0);
    }

    log::debug!("Starting server loop...");
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_io()
        .build()
        .expect("Failed to start tokio");

    runtime.block_on(async move {
        let mut sigterm = signal(SignalKind::terminate()).expect("Failed to install sighandler");

        let accept_task = tokio::spawn(async move {
            let listener = UnixListener::from_std(listener).unwrap();

            log::info!("Listening on unix:{socket_path}");

            while let Ok((conn, _addr)) = listener.accept().await {
                let service = Service {};
                let conn = TokioIo::new(conn);

                if let Err(http_err) = http1::Builder::new()
                    .keep_alive(true)
                    .serve_connection(conn, service)
                    .await
                {
                    log::error!("Error while serving HTTP connection: {http_err}");
                }
            }
        });

        tokio::select! {
            _result = accept_task => {}
            result = sigterm.recv() => {
                if result.is_none() {
                    log::error!("Failed to receive signal. Shutting down.");
                } else {
                    log::info!("Received SIGTERM. Shutting down gracefully...");
                }
            }
        }
    });
}
