#![feature(async_closure) ]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

use std::process::Command;
use std::env::args;
use std::fs::File;
use std::io::Write;
use std::os::unix::io::FromRawFd;

use tokio::net::UnixListener;
use tokio_stream::wrappers::UnixListenerStream;

use nix::unistd::{getpid, fork, ForkResult};
use nix::sched::{CloneFlags, unshare};

#[tokio::main]
async fn main() {
    let make_service = make_service_fn(async move |_| {
        Ok::<_, hyper::Error>(service_fn(async move |req: Request<Body>| {
            log::debug!("Got new request: {:?}", req);

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

    unshare(CloneFlags::CLONE_NEWUTS | CloneFlags::CLONE_NEWPID | CloneFlags::CLONE_NEWIPC).unwrap();

    simple_logging::log_to_file("/host/ol-rust-runtime.log", log::LevelFilter::Trace).unwrap();

    let socket_path = "/host/ol.sock";
    let unix_listener = UnixListener::bind(socket_path).expect("Failed to bind UNIX socket");
    let stream = UnixListenerStream::new(unix_listener);
    let acceptor = hyper::server::accept::from_stream(stream);
    let server = Server::builder(acceptor).serve(make_service);

    log::info!("Listening on unix:{}", socket_path);

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
        log::info!("Joined cgroup, closing FD{}'", fd);
    }

    if let ForkResult::Parent{..} = unsafe{ fork().expect("Fork failed") } {
        std::process::exit(0);
    }

    log::info!("Starting server loop...");

    if let Err(e) = server.await {
        log::error!("server error: {}", e);
    }
}

async fn execute_function(_args: Vec<u8>) -> Result<Response<Body>> {
    log::info!("Executing function");

    Command::new("./f.bin")
        .output()
        .expect("failed to execute lambda function");

    //FIXME
    let body = Body::empty();

    let response = Response::builder()
        .status(StatusCode::OK)
        .body(body)
        .unwrap();

    Ok(response)
}

