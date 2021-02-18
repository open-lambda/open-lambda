#![feature(async_closure) ]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Response, Result, Server, StatusCode};

mod programs;
use programs::ProgramManager;

mod condvar;

mod bindings;
use bindings::get_imports;

use std::sync::Arc;

use wasmer::{Instance, ImportObject};

static mut PROGRAM_MGR: Option<Arc<ProgramManager>> = None;

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let addr = "127.0.0.1:5000".parse().unwrap();
    let programs = Arc::new(ProgramManager::new());
    unsafe{ PROGRAM_MGR = Some(programs) };

    let make_service = make_service_fn(async move |_| {
        Ok::<_, hyper::Error>(service_fn(async move |req| {
            let path = req.uri().path().split("/").filter(|x| x.len() > 0).collect::<Vec<&str>>();
            let args = vec![]; //FIXME
            let program_mgr = unsafe{ PROGRAM_MGR.as_ref().unwrap().clone() };

            if req.method() == &Method::POST && path.len() == 2 && path[0] == "run" {
                execute_function(path[1], args, program_mgr).await
            } else if req.method() == &Method::GET && path.len() == 1 && path[0] == "status" {
                get_status().await
            } else {
                panic!("Got unexpected request: {:?}", req);
            }
        }))
    });

    let server = Server::bind(&addr).serve(make_service);

    log::info!("Listening on http://{}", addr);

    if let Err(e) = server.await {
        log::error!("server error: {}", e);
    }
}

async fn execute_function(name: &str, args: Vec<u8>, program_mgr: Arc<ProgramManager>) -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    let program = program_mgr.get_program(name).await;

    let mut import_object = ImportObject::new();
    import_object.register("open_lambda", get_imports(&*program.store, args));

    let instance = Instance::new(&program.module, &import_object).unwrap();

    let lambda = instance.exports.get_function("f").expect("No function `f` defined");
    lambda.call(&[]).expect("Lambda function failed");

    Ok(response)
}

async fn get_status() -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    Ok(response)
}
