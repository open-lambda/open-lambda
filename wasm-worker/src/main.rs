#![feature(async_closure)]

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

mod functions;
use functions::FunctionManager;

mod condvar;

mod bindings;

mod object_types;

use std::sync::Arc;

use async_wormhole::stack::Stack;

use open_lambda_protocol::{ObjectId, ObjectTypeId};
use lambda_store_utils::WasmCompilerType;

use wasmer::{ImportObject, Instance};

static mut FUNCTION_MGR: Option<Arc<FunctionManager>> = None;

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let addr = "127.0.0.1:5000".parse().unwrap();
    let functions = Arc::new(FunctionManager::new(WasmCompilerType::Cranelift).await);
    unsafe { FUNCTION_MGR = Some(functions) };

    let make_service = make_service_fn(async move |_| {
        Ok::<_, hyper::Error>(service_fn(async move |req: Request<Body>| {
            log::trace!("Got new request: {req:?}");

            let mut path = req
                .uri()
                .path()
                .split('/')
                .filter(|x| !x.is_empty())
                .map(String::from)
                .collect::<Vec<String>>();

            let object_id = req.uri().query()
                .map(ObjectId::from_hex_string);

            let mut args = Vec::new();
            let method = req.method().clone();

            let mut body = req.into_body();

            while let Some(chunk) = body.next().await {
                match chunk {
                    Ok(c) => {
                        let mut chunk = c.to_vec();
                        args.append(&mut chunk);
                    }
                    Err(err) => {
                        panic!("Got error: {err:?}");
                    }
                }
            }

            let function_mgr = unsafe { FUNCTION_MGR.as_ref().unwrap().clone() };

            if method == Method::POST && path.len() == 2 && path[0] == "run" {
                execute_function(object_id.expect("No object id given"), path.pop().unwrap(), args, function_mgr).await
            } else if method == Method::GET && path.len() == 1 && path[0] == "status" {
                get_status().await
            } else {
                panic!("Got unexpected request to {path:?} (Method: {method:?})");
            }
        }))
    });

    let server = Server::bind(&addr).serve(make_service);

    log::info!("Listening on http://{addr}");

    if let Err(err) = server.await {
        log::error!("server error: {err}");
    }
}

async fn execute_function(
    object_id: ObjectId,
    name: String,
    args: Vec<u8>,
    function_mgr: Arc<FunctionManager>,
) -> Result<Response<Body>> {
    let db = Arc::new(lambda_store_client::create_client("localhost").await
        .expect("Failed to set up client"));
    let object= db.get_object(object_id).await.expect("No such object");

    let functions = function_mgr.get_object_functions(&object.get_type_id()).await
        .expect("Binary for object type is missing");
    let args = Arc::new(args);

    loop {
        let result = Arc::new(std::sync::Mutex::new(None));
        let storage = bindings::storage::StorageEnv::new(db.clone());

        let instance = {
            let mut import_object = ImportObject::new();
            import_object.register(
                "ol_args",
                bindings::args::get_imports(&*functions.store, args.clone(), result.clone()),
            );
            import_object.register("ol_log", bindings::log::get_imports(&*functions.store));
            import_object.register(
                "ol_storage",
                bindings::storage::get_imports(&*functions.store, storage.clone()),
            );

            Instance::new(&functions.module, &import_object).unwrap()
        };

        let stack = async_wormhole::stack::EightMbStack::new().unwrap();
        if let (Err(e), _) = instance.call_with_stack("f", stack, vec![object_id.into_int()]).await {
            if let Some(wasmer_vm::TrapCode::StackOverflow) = e.clone().to_trap() {
                log::error!("Function failed due to stack overflow");
            } else {
                log::error!("Function failed with message \"{}\"", e.message());
                log::error!("Stack trace:");

                for frame in e.trace() {
                    log::error!(
                        "   {}::{}",
                        frame.module_name(),
                        frame.function_name().or(Some("unknown")).unwrap()
                    );
                }
            }
        };

        if storage.commit().await {
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

            return Ok(response);
        }
    }
}

async fn get_status() -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    Ok(response)
}
