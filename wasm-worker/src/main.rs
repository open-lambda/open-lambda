#![feature(type_alias_impl_trait)]

use std::fs::{read_dir, remove_file, File};
use std::net::{SocketAddr, ToSocketAddrs};
use std::path::PathBuf;

use serde::{Deserialize, Serialize};

use http_body_util::{BodyExt, Full};

use hyper::server::conn::http1;
use hyper::body::{Bytes, Incoming};
use hyper::{http, Method, Request, Response, StatusCode};

use tokio::signal::unix::{signal, SignalKind};

mod functions;
use functions::FunctionManager;

use parking_lot::Mutex;

mod bindings;

mod http_client;

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

    #[clap(long)]
    #[clap(help = "Address of the lambdastore coordinator")]
    lambdastore_coord: Option<String>,
}

async fn load_functions(registry_path: &str, function_mgr: &Arc<FunctionManager>) {
    let compiler_name = format!("{}", function_mgr.get_compiler_type()).to_lowercase();
    let cache_path: PathBuf = format!("{registry_path}.{compiler_name}.cache").into();

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

struct Service {
    worker_addr: SocketAddr,
    function_mgr: Arc<FunctionManager>,
}

impl hyper::service::Service<Request<Incoming>> for Service {
    type Response = Response<Full<Bytes>>;
    type Error = http::Error;
    type Future = impl std::future::Future<Output = http::Result<Response<Full<Bytes>>>>;

    fn call(&mut self, req: Request<Incoming>) -> Self::Future {
        Self::handle_request(req, self.worker_addr, self.function_mgr.clone())
    }
}

impl Service {
    async fn handle_request(req: Request<Incoming>, worker_addr: SocketAddr, function_mgr: Arc<FunctionManager>) -> http::Result<Response<Full<Bytes>>> {
        log::trace!("Got new request: {req:?}");

        let mut path = req
            .uri()
            .path()
            .split('/')
            .filter(|x| !x.is_empty())
            .map(String::from)
            .collect::<Vec<String>>();

        let method = req.method().clone();
        let args = req.collect().await.unwrap().to_bytes().to_vec();

        if method == Method::POST && path.len() == 2 && path[0] == "run" {
            Self::execute_function(worker_addr, &path.pop().unwrap(), args, function_mgr)
                .await
        } else if method == Method::GET && path.len() == 1 && path[0] == "status" {
            Self::get_status().await
        } else {
            panic!("Got unexpected request to {path:?} (Method: {method:?})");
        }
    }

    async fn execute_function(
        worker_addr: SocketAddr,
        name: &str,
        args: Vec<u8>,
        function_mgr: Arc<FunctionManager>,
    ) -> http::Result<Response<Full<Bytes>>> {
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
            vec![].into()
        };

        let response = Response::builder()
            .status(StatusCode::OK)
            .body(body)?;

        instance.mark_idle();
        Ok(response)
    }

    async fn get_status() -> http::Result<Response<Full<Bytes>>> {
        Response::builder()
            .status(StatusCode::OK)
            .body(vec![].into())
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

    if let Some(coord_address) = args.lambdastore_coord {
        cfg_if::cfg_if!{
            if #[cfg(feature="lambdastore")] {
                let result = unsafe {
                    bindings::extra::lambdastore::create_client(&coord_address).await
                };

                if let Err(err) = result {
                    log::error!("Failed to setup lambdastore module: {err}");
                    return;
                }
            } else {
                panic!("`lambdastore`-feature not enabled");
            }
        }
    }

    let listener = tokio::net::TcpListener::bind(&worker_addr)
        .await
        .expect("Failed to bind socket for frontend");

    let fut = tokio::spawn(async move {
        while let Ok((conn, addr)) = listener.accept().await {
            log::debug!("Got new connection from {addr}");

            let function_mgr = function_mgr.clone();

            tokio::spawn(async move {
                conn.set_nodelay(true).unwrap();
                let service = Service { worker_addr, function_mgr };

                if let Err(http_err) = http1::Builder::new()
                    .keep_alive(true)
                    .serve_connection(conn, service)
                    .await
                {
                    log::error!("Error while serving HTTP connection: {http_err}");
                }
            });
        }
    });

    log::info!("Listening on http://{worker_addr}");

    let mut sigterm = signal(SignalKind::terminate()).expect("Failed to install sighandler");
    let mut sigint = signal(SignalKind::interrupt()).expect("Failed to install sighandler");

    File::create("./ol-wasm.ready").expect("Failed to create ready file");

    tokio::select! {
        result = fut => {
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


