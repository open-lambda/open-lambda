#![feature(async_closure) ]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

use std::process::Stdio;
use std::env::args;
use std::fs::File;
use std::io::Write;
use std::os::unix::io::FromRawFd;

use tokio::io::AsyncReadExt;
use tokio::select;
use tokio::process::Command;
use tokio::net::UnixListener;
use tokio_stream::wrappers::UnixListenerStream;
use tokio::signal::unix::{signal, SignalKind};

use nix::unistd::{getpid, fork, ForkResult};
use nix::sched::{CloneFlags, unshare};

use std::os::unix::net::UnixListener as StdUnixListener;

// Taken from: https://github.com/hyperium/hyper/blob/master/examples/single_threaded.rs
#[derive(Clone, Copy, Debug)]
struct LocalExec;

impl<F> hyper::rt::Executor<F> for LocalExec
where
    F: std::future::Future + 'static, // not requiring `Send`
{
    fn execute(&self, fut: F) {
        // This will spawn into the currently running `LocalSet`.
        tokio::task::spawn_local(fut);
    }
}

fn main() {
    let make_service = make_service_fn(async move |_| {
        Ok::<_, hyper::Error>(service_fn(async move |req: Request<Body>| {
            log::trace!("Got new request: {:?}", req);

            let mut args = Vec::new();
            let method = req.method().clone();

            let mut body = req.into_body();

            while let Some(chunk) = body.next().await {
                match chunk {
                    Ok(c) => {
                        let mut chunk = c.to_vec();
                        args.append(&mut chunk);
                    }
                    Err(e) => {
                        panic!("Got error: {:?}", e);
                    }
                }
            }

            if method == &Method::POST {
                execute_function(args).await
            } else {
                panic!("Got unexpected request");
            }
        }))
    });

    if let Err(err) = simple_logging::log_to_file("/host/ol-runtime.log", log::LevelFilter::Info) {
        println!("Failed to create logfile: {}", err);
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
        let mut f = unsafe{ File::from_raw_fd(fd) };

        f.write(format!("{}", pid).as_bytes()).unwrap();
        log::trace!("Joined cgroup, closing FD{}'", fd);
    }

    unshare(CloneFlags::CLONE_NEWUTS | CloneFlags::CLONE_NEWPID | CloneFlags::CLONE_NEWIPC).unwrap();

    if let ForkResult::Parent{..} = unsafe{ fork().expect("Fork failed") } {
        std::process::exit(0);
    }

    log::info!("Starting server loop...");
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_io().build().expect("Failed to start tokio");

    runtime.block_on(async move {
        let mut sighandler = signal(SignalKind::terminate()).expect("Failed to install sighandler");

        let listener = UnixListener::from_std(listener).unwrap();
        let stream = UnixListenerStream::new(listener);
        let acceptor = hyper::server::accept::from_stream(stream);

        let server = Server::builder(acceptor).serve(make_service)
            .with_graceful_shutdown(async move {
                sighandler.recv().await;
                log::info!("Got ctrl+c");
            });

        log::info!("Listening on unix:{}", socket_path);

        if let Err(e) = server.await {
            log::error!("server error: {}", e);
        }
    });
}

async fn execute_function(args: Vec<u8>) -> Result<Response<Body>> {
    use std::io::Read;

    let arg_str = String::from_utf8(args).unwrap();
    log::info!("Executing function with arg `{}`", arg_str);

    let body;
    let status_code;

    let mut child = Command::new("/handler/f.bin").arg(arg_str)
            .env("RUST_LOG", "debug")
            .stdout(Stdio::piped()).stderr(Stdio::piped())
            .spawn().expect("Failed to spawn lambda process");

    let child_future = child.wait();

    let mut sighandler = signal(SignalKind::terminate()).expect("Failed to install sighandler");
    let sig_future = sighandler.recv();

    let mut e_str = String::from("");

    log::debug!("Waiting for process to terminate or signal");

    let success = select! {
        _ = sig_future => {
            log::info!("Got ctrl+c");
            child.kill().await.expect("Failed to kill child");
            child.wait().await.unwrap();

            std::process::exit(0);
        }
        res = child_future => {
            let status = res.unwrap();

            if status.success() {
                true
            } else {
                e_str = format!("Function failed with exitcode: {}", status.code().unwrap());
                false
            }
        }
    };

    if !success {
        status_code = StatusCode::INTERNAL_SERVER_ERROR;
        log::error!("{}", e_str);
        body = e_str.into();
    } else {
        log::info!("Function returned successfully");

        match File::open("/tmp/output") {
            Ok(mut f) => {
                let mut jstr = String::new();
                f.read_to_string(&mut jstr).unwrap();

                log::info!("Got response: {}", jstr);

                body = Body::from(jstr);
                status_code = StatusCode::OK;
            }
            Err(e) => {
                if e.kind() !=  std::io::ErrorKind::NotFound {
                    let e_str = format!("Got unexpected error: {:?}", e);
                    log::error!("{}", e_str);

                    body = Body::from(e_str);
                    status_code = StatusCode::INTERNAL_SERVER_ERROR;
                } else {
                    log::info!("Function did not give a response");

                    body = Body::empty();
                    status_code = StatusCode::OK;
                }
            }
        }
    }

    let mut stdout = String::from("");
    let mut stderr = String::from("");

    child.stdout.take().expect("Failed to get child stdout")
        .read_to_string(&mut stdout).await.unwrap();
    child.stderr.take().expect("Failed to get child stderr")
        .read_to_string(&mut stderr).await.unwrap();

    if stdout == "" {
        log::debug!("Program has no stdout output");
    } else {
        let mut log_line = String::from("Process stdout:\n");

        for line in stdout.split('\n') {
            log_line += format!("    {}\n", line).as_str();
        }

        log::info!("{}", log_line);
    }

    if stderr == "" {
        log::debug!("Program has no stderr output");
    } else {
        let mut log_line = String::from("Process stderr:\n");

        for line in stderr.split('\n') {
            log_line += format!("    {}\n", line).as_str();
        }

        log::info!("{}", log_line);
    }

    let response = Response::builder()
        .status(status_code)
        .body(body)
        .unwrap();

    Ok(response)
}
