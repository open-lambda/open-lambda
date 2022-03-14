#![feature(async_closure)]

use std::net::SocketAddr;
use std::path::{Path, PathBuf};

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

use percent_encoding::percent_decode_str;

mod functions;
use functions::FunctionManager;

mod bindings;

use std::sync::Arc;

use clap::Parser;

use async_wormhole::stack::Stack;

use lambda_store_utils::WasmCompilerType;
use open_lambda_protocol::ObjectId;

use wasmer::{ImportObject, Instance};

static mut FUNCTION_MGR: Option<Arc<FunctionManager>> = None;
static mut LS_ADDRESS: Option<String> = None;

#[derive(Parser)]
#[clap(rename_all = "snake-case")]
#[clap(author, version, about, long_about = None)]
struct Args {
    #[clap(long, arg_enum, default_value = "llvm")]
    #[clap(help = "Which compiler should be used to compile WebAssembly to native code?")]
    wasm_compiler: WasmCompilerType,

    #[clap(long, short='c', default_value = "localhost")]
    #[clap(help = "What is the address of the lambda store coordinator?")]
    coordinator_address: String,

    #[clap(long, short='l', default_value = "localhost:5000")]
    #[clap(help = "What is the address to listen on for client requests?")]
    listen_address: String,
}

async fn load_functions(args: &Args, function_mgr: &Arc<FunctionManager>) {
    let registry_path = "test-registry.wasm";
    let compiler_name = format!("{}", function_mgr.get_compiler_type()).to_lowercase();
    let cache_path: PathBuf = format!("{registry_path}.worker.{compiler_name}.cache").into();

    let db = lambda_store_client::create_client(&args.coordinator_address)
        .await
        .expect("Failed to connect to database");

    for (type_id, name, _object_type) in db.get_object_types() {
        if name == "root" {
            // ignore root type
            continue;
        }

        let file_name = format!("{registry_path}/{name}.wasm");

        function_mgr
            .load_object_functions(
                type_id,
                Path::new(&file_name).to_path_buf(),
                cache_path.clone(),
            )
            .await;
    }

    db.close().await;
}

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let args = Args::parse();

    let worker_addr: SocketAddr = args.listen_address.parse().unwrap();

    let function_mgr = Arc::new(FunctionManager::new(args.wasm_compiler).await);

    load_functions(&args, &function_mgr).await;

    unsafe {
        FUNCTION_MGR = Some(function_mgr);
        LS_ADDRESS = Some(args.coordinator_address);
    }

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

            let object_id = if let Some(query) = req.uri().query() {
                let mut split = query.split('=');
                if split.next().unwrap() == "object_id" {
                    let oid = percent_decode_str(split.next().unwrap())
                        .decode_utf8()
                        .unwrap();
                    Some(ObjectId::from_hex_string(&*oid))
                } else {
                    None
                }
            } else {
                None
            };

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
            let ls_addr = unsafe { LS_ADDRESS.as_ref().unwrap().clone() };

            if method == Method::POST && path.len() == 2 && path[0] == "run_on" {
                execute_function(
                    worker_addr, &ls_addr,
                    object_id.expect("No object id given"),
                    &path.pop().unwrap(),
                    args,
                    function_mgr,
                )
                .await
            } else if method == Method::GET && path.len() == 1 && path[0] == "status" {
                get_status().await
            } else {
                panic!("Got unexpected request to {path:?} (Method: {method:?})");
            }
        }))
    });

    let server = Server::bind(&worker_addr).serve(make_service);

    log::info!("Listening on http://{worker_addr}");

    if let Err(err) = server.await {
        log::error!("server error: {err}");
    }
}

async fn execute_function(
    worker_addr: SocketAddr,
    ls_addr: &str,
    object_id: ObjectId,
    name: &str,
    args: Vec<u8>,
    function_mgr: Arc<FunctionManager>,
) -> Result<Response<Body>> {
    let db = Arc::new(
        lambda_store_client::create_client(&ls_addr)
            .await
            .expect("Failed to set up client"),
    );
    let object = db.get_object(object_id).await.expect("No such object");

    let functions = function_mgr
        .get_object_functions(&object.get_type_id())
        .await
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
            import_object.register(
                "ol_ipc",
                bindings::ipc::get_imports(&*functions.store, worker_addr, db.clone()),
            );
            import_object.register("ol_log", bindings::log::get_imports(&*functions.store));
            import_object.register(
                "ol_storage",
                bindings::storage::get_imports(&*functions.store, storage.clone()),
            );

            Instance::new(&functions.module, &import_object).unwrap()
        };

        let stack = async_wormhole::stack::EightMbStack::new().unwrap();
        if let (Err(e), _) = instance
            .call_with_stack(name, stack, vec![object_id.into_int()])
            .await
        {
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
