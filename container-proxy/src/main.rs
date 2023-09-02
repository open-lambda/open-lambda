use std::fs::File;
use std::io::Write;

use tokio::net::{UnixListener, UnixStream};

use futures_util::sink::SinkExt;
use futures_util::stream::StreamExt;

use tokio_util::codec::length_delimited::LengthDelimitedCodec;
use tokio_util::codec::{FramedRead, FramedWrite};

use serde_bytes::ByteBuf;

use open_lambda_proxy_protocol::ProxyMessage;

use tokio_uring_executor as executor;

mod extra;

fn main() {
    executor::initialize();

    //FIXME this adds an additional executor thread
    tokio_uring::start(async move {
        main_logic().await;
    });
}

async fn main_logic() {
    let mut argv = std::env::args();
    argv.next().unwrap();

    let container_dir = argv.next().expect("No container directory given");

    #[cfg(feature="lambdastore")]
    extra::lambdastore::set_address(argv.next().expect("No lambdastore path given"));

    simple_logging::log_to_file(
        format!("{container_dir}/container-proxy.log"),
        log::LevelFilter::Info,
    )
    .unwrap();

    let path = format!("{container_dir}/proxy.sock");
    let listener = UnixListener::bind(path).unwrap();

    // Create pid file to notify others that the socket is bound
    let mut file = File::create(format!("{container_dir}/proxy.pid")).unwrap();

    let content = format!("{}", std::process::id()).into_bytes();
    file.write_all(&content).unwrap();

    loop {
        let accept = listener.accept();
        let signal = tokio::signal::ctrl_c();

        tokio::select! {
            _ = signal => {
                log::info!("Got ctrl+c");
                std::process::exit(0);
            }
            result = accept => {
                match result {
                    Ok((stream, _)) => unsafe {
                        executor::unsafe_spawn(async move {
                            handle_connection(stream).await;
                        });
                    }
                    Err(err) => {
                        log::error!("Got error: {err}");
                        std::process::exit(-1);
                    }
                }
            }
        }
    }
}

async fn function_call(func_name: String, args: Vec<u8>) -> Result<Vec<u8>, String> {
    log::trace!("Issuing function call to {func_name}");

    let server_addr = "localhost:5000";
    let url = format!("http://{server_addr}/run/{func_name}");
    let client = reqwest::ClientBuilder::new()
        .tcp_nodelay(true)
        .build()
        .expect("Failed to set up HTTP client");

    let request = client.post(url).body(args);

    let result = match request.send().await {
        Ok(result) => result,
        Err(err) => {
            return Err(format!("Failed to send call request: {err}"));
        }
    };

    if result.status().is_success() {
        match result.bytes().await {
            Ok(bytes) => {
                let mut data = vec![];
                data.extend_from_slice(&bytes[..]);
                Ok(data)
            }
            Err(err) => Err(format!("Failed to process call result: {err}")),
        }
    } else {
        Err(format!(
            "Call Request failed with error code: {}",
            result.status()
        ))
    }
}

async fn handle_connection(stream: UnixStream) {
    log::debug!("Connected to process");

    let (reader, writer) = stream.into_split();

    let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
    let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

    while let Some(res) = reader.next().await {
        let data = match res {
            Ok(data) => data,
            Err(err) => {
                log::error!("Failed to receive data from runtime: {err}");
                break;
            }
        };

        let msg = bincode::deserialize(&data).unwrap();

        let response = match msg {
            ProxyMessage::FuncCallRequest(call_data) => {
                let result = function_call(call_data.fn_name, call_data.args.into_vec()).await;
                ProxyMessage::FuncCallResult(result.map(ByteBuf::from))
            }
            ProxyMessage::HostCallRequest(call_data) => {
                let result = if call_data.namespace == "lambdastore" {
                    cfg_if::cfg_if! {
                        if #[ cfg(feature="lambdastore") ] {
                            crate::extra::lambdastore::call(&call_data.fn_name, &call_data.args).await
                        } else {
                            panic!("Feature `lambdastore` not enabled");
                        }
                    }
                } else {
                    panic!("Unknown host call namespace: {}", call_data.namespace);
                };

                ProxyMessage::HostCallResult(result)
            }
            _ => {
                panic!("Got unexpected message: {msg:?}");
            }
        };

        let out_data = bincode::serialize(&response).unwrap();
        writer
            .send(out_data.into())
            .await
            .expect("Failed to send response");
    }
}
