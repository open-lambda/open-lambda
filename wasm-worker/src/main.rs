#![feature(impl_trait_in_assoc_type)]

use std::collections::HashMap;
use std::fs::{File, read_dir, remove_file};
use std::net::{SocketAddr, ToSocketAddrs};
use std::path::PathBuf;
use std::sync::Arc;
use std::thread::available_parallelism;

use http_body_util::{BodyExt, Full};

use hyper::body::{Bytes, Incoming};
use hyper::server::conn::http1;
use hyper::{Method, Request, Response, StatusCode, http};

use tokio::runtime;
use tokio::signal::unix::{SignalKind, signal};

use anyhow::Context;

use clap::Parser;

mod support;

mod functions;
use functions::FunctionManager;

use parking_lot::Mutex;

mod bindings;

mod http_client;

#[derive(Parser)]
#[clap(author, version, about, long_about = None)]
struct Args {
    #[clap(long, short = 'l', default_value = "localhost:5000")]
    #[clap(help = "What is the address to listen on for client requests?")]
    listen_address: String,

    #[clap(long, short = 'p', default_value = "./test-registry.wasm")]
    #[clap(help = "Where are the WASM functions stored?")]
    registry_path: String,

    #[clap(long)]
    enable_cpu_profiler: bool,

    #[clap(short = 'C')]
    config_values: Option<Vec<String>>,
}

async fn load_functions(
    registry_path: &str,
    function_mgr: &Arc<FunctionManager>,
) -> anyhow::Result<()> {
    let cache_path: PathBuf = format!("{registry_path}.cache").into();

    let directory = match read_dir(registry_path) {
        Ok(dir) => dir,
        Err(err) => {
            anyhow::bail!("Failed to open registry at {registry_path:?}: {err}");
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

    Ok(())
}

struct Service {
    worker_addr: SocketAddr,
    function_mgr: Arc<FunctionManager>,
    config_values: Arc<HashMap<String, String>>,
}

impl hyper::service::Service<Request<Incoming>> for Service {
    type Response = Response<Full<Bytes>>;
    type Error = http::Error;
    type Future = impl std::future::Future<Output = http::Result<Response<Full<Bytes>>>>;

    fn call(&self, req: Request<Incoming>) -> Self::Future {
        Self::handle_request(
            req,
            self.worker_addr,
            self.function_mgr.clone(),
            self.config_values.clone(),
        )
    }
}

impl Service {
    async fn handle_request(
        req: Request<Incoming>,
        worker_addr: SocketAddr,
        function_mgr: Arc<FunctionManager>,
        config_values: Arc<HashMap<String, String>>,
    ) -> http::Result<Response<Full<Bytes>>> {
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
            Self::execute_function(
                worker_addr,
                &path.pop().unwrap(),
                args,
                function_mgr,
                config_values,
            )
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
        config_values: Arc<HashMap<String, String>>,
    ) -> http::Result<Response<Full<Bytes>>> {
        let result = Arc::new(Mutex::new(None));

        let function = match function_mgr.get_function(name).await {
            Some(func) => func,
            None => panic!("No such function \"{name}\""),
        };

        log::trace!("Starting function call for \"{name}\"");

        let mut instance_hdl = function
            .get_idle_instance(args, &config_values, worker_addr, result.clone())
            .await;

        let (mut store, instance) = instance_hdl.get();

        let call_result = instance
            .get_func(&mut store, "f")
            .unwrap()
            .call_async(store, &[], &mut [])
            .await;

        let response = if let Err(error) = call_result {
            // Handle a regular crash here
            log::error!("Function failed with message \"{}\"", error.root_cause());

            let response = Response::builder()
                .status(StatusCode::INTERNAL_SERVER_ERROR)
                .body(Default::default())?;
            instance_hdl.mark_idle();

            response
        } else {
            let result = result.lock().take();

            let body = if let Some(result) = result {
                result.into()
            } else {
                vec![].into()
            };

            let response = Response::builder().status(StatusCode::OK).body(body)?;

            instance_hdl.mark_idle();
            log::trace!("Done with function call for \"{name}\"");

            response
        };

        Ok(response)
    }

    async fn get_status() -> http::Result<Response<Full<Bytes>>> {
        Response::builder()
            .status(StatusCode::OK)
            .body(vec![].into())
    }
}

async fn main_func(args: Args) -> anyhow::Result<()> {
    let worker_addr: SocketAddr = match args.listen_address.to_socket_addrs() {
        Ok(mut addrs) => addrs.next().unwrap(),
        Err(err) => {
            anyhow::bail!(
                "Failed to parse listen address \"{}\": {err}",
                args.listen_address
            );
        }
    };

    let function_mgr = Arc::new(
        FunctionManager::new()
            .await
            .with_context(|| "Failed to create function manager")?,
    );

    let mut config_values = HashMap::default();

    if let Some(vals) = args.config_values {
        for entry in vals {
            let mut split = entry.split('=');
            let key = split.next().expect("Invalid config setting");
            let value = split.next().expect("Invalid config setting");

            config_values.insert(key.to_string(), value.to_string());
        }
    }

    let config_values = Arc::new(config_values);

    load_functions(&args.registry_path, &function_mgr).await?;

    let listener = tokio::net::TcpListener::bind(&worker_addr)
        .await
        .unwrap_or_else(|err| {
            panic!("Failed to bind socket for OL wasm-worker at {worker_addr}: {err}")
        });

    #[cfg(feature = "cpuprofiler")]
    let enable_cpu_profiler = args.enable_cpu_profiler;

    #[cfg(feature = "cpuprofiler")]
    if enable_cpu_profiler {
        let mut profiler = cpuprofiler::PROFILER.lock().unwrap();
        let fname = format!("ol-wasm_{}.profile", std::process::id());
        profiler
            .start(fname.clone())
            .expect("Failed to start profiler");

        log::info!("CPU profiler enabled. Writing output to '{fname}'");
    }

    let fut = tokio::spawn(async move {
        while let Ok((conn, addr)) = listener.accept().await {
            log::debug!("Got new connection from {addr}");

            let function_mgr = function_mgr.clone();
            let config_values = config_values.clone();

            tokio::spawn(async move {
                let service = Service {
                    worker_addr,
                    function_mgr,
                    config_values,
                };

                conn.set_nodelay(true).unwrap();
                let conn = support::TokioIo::new(conn);

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

    #[cfg(feature = "cpuprofiler")]
    if enable_cpu_profiler {
        let mut profiler = cpuprofiler::PROFILER.lock().unwrap();
        profiler.stop().expect("Failed to stop profiler");
    }

    remove_file("./ol-wasm.ready").unwrap();

    Ok(())
}

fn main() -> anyhow::Result<()> {
    env_logger::init();

    let args = Args::parse();

    let num_threads = 2 * available_parallelism().unwrap().get();

    let rt = runtime::Builder::new_multi_thread()
        .enable_io()
        .worker_threads(num_threads)
        .build()
        .unwrap();

    rt.block_on(async move { main_func(args).await })
}
