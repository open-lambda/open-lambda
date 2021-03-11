#![feature(async_closure) ]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

use std::process::Command;

use tokio::net::UnixListener;
use tokio_stream::wrappers::UnixListenerStream;

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

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

    let socket_path = "/host/ol.sock";
    let unix_listener = UnixListener::bind(socket_path).expect("Failed to bind UNIX socket");
    let stream = UnixListenerStream::new(unix_listener);
    let acceptor = hyper::server::accept::from_stream(stream);
    let server = Server::builder(acceptor).serve(make_service);

    log::info!("Listening on unix://{}", socket_path);

    if let Err(e) = server.await {
        log::error!("server error: {}", e);
    }
}

async fn execute_function(_args: Vec<u8>) -> Result<Response<Body>> {
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

