#![feature(async_closure) ]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

mod programs;
use programs::ProgramManager;

mod condvar;

mod bindings;

use std::sync::Arc;

use async_wormhole::stack::Stack;

use wasmer::{Instance, ImportObject};

static mut PROGRAM_MGR: Option<Arc<ProgramManager>> = None;

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let addr = "127.0.0.1:5000".parse().unwrap();
    let programs = Arc::new(ProgramManager::new());
    unsafe{ PROGRAM_MGR = Some(programs) };

    let make_service = make_service_fn(async move |_| {
        Ok::<_, hyper::Error>(service_fn(async move |req: Request<Body>| {
            log::trace!("Got new request: {:?}", req);

            let mut path = req.uri().path().split("/").filter(|x| x.len() > 0)
                .map(|x| String::from(x)).collect::<Vec<String>>();

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

            let program_mgr = unsafe{ PROGRAM_MGR.as_ref().unwrap().clone() };

            if method == &Method::POST && path.len() == 2 && path[0] == "run" {
                execute_function(path.pop().unwrap(), args, program_mgr).await
            } else if  method == &Method::GET && path.len() == 1 && path[0] == "status" {
                get_status().await
            } else {
                panic!("Got unexpected request to {:?} (Method: {:?})", path, method);
            }
        }))
    });

    let server = Server::bind(&addr).serve(make_service);

    log::info!("Listening on http://{}", addr);

    if let Err(e) = server.await {
        log::error!("server error: {}", e);
    }
}

async fn execute_function(name: String, args: Vec<u8>, program_mgr: Arc<ProgramManager>) -> Result<Response<Body>> {
    let program = program_mgr.get_program(name).await;
    let result = Arc::new(std::sync::Mutex::new(None));

    let instance = {
        let mut import_object = ImportObject::new();
        import_object.register("ol_args", crate::bindings::args::get_imports(&*program.store, args, result.clone()));
        import_object.register("ol_log", crate::bindings::log::get_imports(&*program.store));
        import_object.register("ol_storage", bindings::storage::get_imports(&*program.store));

        Instance::new(&program.module, &import_object).unwrap()
    };

    let stack = async_wormhole::stack::EightMbStack::new().unwrap();
    if let Err(e) = instance.call_with_stack("f", stack).await {
        if let Some(wasmer_vm::TrapCode::StackOverflow) = e.clone().to_trap() {
            log::error!("Function failed due to stack overflow");
        } else {
            log::error!("Function failed with message \"{}\"", e.message());
            log::error!("Stack trace:");

            for frame in e.trace() {
                log::error!("   {}::{}", frame.module_name(), frame.function_name().or(Some("unknown")).unwrap());
            }
        }
    };

    let result = result.lock().unwrap().take();

    let body = if let Some(result) = result {
        result.into()
    } else {
        Body::empty()
    };

    let response = Response::builder()
        .status(StatusCode::OK)
        .body(body)
        .unwrap();

    Ok(response)
}

async fn get_status() -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    Ok(response)
}
