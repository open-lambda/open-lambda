use std::fs::{read_dir, remove_file, File};
use std::net::{SocketAddr, ToSocketAddrs};
use std::path::PathBuf;

use serde::{Deserialize, Serialize};

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

use futures_util::stream::StreamExt;

use tokio::signal::unix::{signal, SignalKind};

mod functions;
use functions::FunctionManager;

use parking_lot::Mutex;

mod bindings;

use std::sync::Arc;

use clap::Parser;

use async_wormhole::stack::Stack;

#[derive(
    Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize, derive_more::Display, clap::ValueEnum,
)]
pub enum WasmCompilerType {
    LLVM,
    Cranelift,
    Singlepass,
}

#[derive(Parser)]
#[clap(author, version, about, long_about = None)]
struct Args {
    #[clap(long, value_enum, default_value = "llvm")]
    #[clap(help = "Which compiler should be used to compile WebAssembly to native code?")]
    wasm_compiler: WasmCompilerType,

    #[clap(long, short = 'l', default_value = "localhost:5000")]
    #[clap(help = "What is the address to listen on for client requests?")]
    listen_address: String,

    #[clap(long, short = 'p', default_value = "./test-registry.wasm")]
    #[clap(help = "Where are the WASM functions stored?")]
    registry_path: String,
}

async fn load_functions(registry_path: &str, function_mgr: &Arc<FunctionManager>) {
    let compiler_name = format!("{}", function_mgr.get_compiler_type()).to_lowercase();
    let cache_path: PathBuf = format!("{registry_path}.worker.{compiler_name}.cache").into();

    let directory = match read_dir(registry_path) {
        Ok(dir) => dir,
        Err(err) => {
            panic!("Failed to open registry at {registry_path:?}: {err}");
        }
    };

    for entry in directory {
        let entry = entry.expect("Failed to read next file");
        let file_path = entry.path();

        if !entry.file_type().unwrap().is_file() {
            log::warn!("Entry {file_path:?} is not a regular file. Skipping...");
            continue;
        }

        let extension = match file_path.extension() {
            Some(ext) => ext,
            None => {
                log::warn!("Entry {file_path:?} does not have a file extension. Skipping...");
                continue;
            }
        };

        if extension != "wasm" {
            log::warn!("Entry {file_path:?} is not a WebAssembly file. Skipping...");
            continue;
        }

        function_mgr
            .load_function(file_path, cache_path.clone())
            .await;
    }
}

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let args = Args::parse();

    let worker_addr: SocketAddr = match args.listen_address.to_socket_addrs() {
        Ok(mut addrs) => addrs.next().unwrap(),
        Err(err) => {
            log::error!(
                "Failed to parse listen address \"{}\": {err}",
                args.listen_address
            );
            return;
        }
    };

    let function_mgr = Arc::new(FunctionManager::new(args.wasm_compiler).await);

    load_functions(&args.registry_path, &function_mgr).await;

    let make_service = make_service_fn(move |_| {
        let function_mgr = function_mgr.clone();

        async move {
            Ok::<_, hyper::Error>(service_fn(move |req: Request<Body>| {
                let function_mgr = function_mgr.clone();

                async move {
                    log::trace!("Got new request: {req:?}");

                    let mut path = req
                        .uri()
                        .path()
                        .split('/')
                        .filter(|x| !x.is_empty())
                        .map(String::from)
                        .collect::<Vec<String>>();

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

                    if method == Method::POST && path.len() == 2 && path[0] == "run" {
                        execute_function(worker_addr, &path.pop().unwrap(), args, function_mgr)
                            .await
                    } else if method == Method::GET && path.len() == 1 && path[0] == "status" {
                        get_status().await
                    } else {
                        panic!("Got unexpected request to {path:?} (Method: {method:?})");
                    }
                }
            }))
        }
    });

    let server = Server::bind(&worker_addr).serve(make_service);

    log::info!("Listening on http://{worker_addr}");

    let mut sigterm = signal(SignalKind::terminate()).expect("Failed to install sighandler");
    let mut sigint = signal(SignalKind::interrupt()).expect("Failed to install sighandler");

    File::create("./ol-wasm.ready").expect("Failed to create ready file");

    tokio::select! {
        result = server => {
            if let Err(err) = result {
                log::error!("Got server error: {err}");
            }
        }
        result = sigterm.recv() => {
            if result.is_none() {
                log::error!("Failed to receive signal. Shutting down.");
            } else {
                log::info!("Received SIGTERM. Shutting down gracefully...");
            }
        },
        result = sigint.recv() => {
            if result.is_none() {
                log::error!("Failed to receive signal. Shutting down.");
            } else {
                log::info!("Received SIGINT. Shutting down gracefully...");
            }
        }
    }

    remove_file("./ol-wasm.ready").unwrap();
}

async fn execute_function(
    worker_addr: SocketAddr,
    name: &str,
    args: Vec<u8>,
    function_mgr: Arc<FunctionManager>,
) -> Result<Response<Body>> {
    let args = Arc::new(args);

    let result = Arc::new(Mutex::new(None));

    let function = match function_mgr.get_function(name).await {
        Some(func) => func,
        None => panic!("No such function \"{name}\""),
    };

    let instance = function.get_idle_instance(args.clone(), worker_addr, result.clone());

    let func_args: Vec<u32> = vec![];

    let stack = async_wormhole::stack::EightMbStack::new().unwrap();
    if let (Err(e), _) = instance.get().call_with_stack("f", stack, func_args).await {
        if let Some(wasmer_vm::TrapCode::StackOverflow) = e.clone().to_trap() {
            log::error!("Function failed due to stack overflow");
        } else {
            log::error!("Function failed with message \"{}\"", e.message());
            log::error!("Stack trace:");

            for frame in e.trace() {
                log::error!(
                    "   {}::{}",
                    frame.module_name(),
                    frame.function_name().unwrap_or("unknown")
                );
            }
        }
    };

    let result = result.lock().take();

    let body = if let Some(result) = result {
        result.into()
    } else {
        Body::empty()
    };

    let response = Response::builder()
        .status(StatusCode::OK)
        .body(body)
        .unwrap();

    instance.mark_idle();
    Ok(response)
}

async fn get_status() -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    Ok(response)
}
