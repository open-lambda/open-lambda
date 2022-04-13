use std::fs::File;
use std::io::Write;

use tokio::net::{UnixListener, UnixStream};

use futures_util::stream::StreamExt;
use futures_util::sink::SinkExt;

use tokio_util::codec::{FramedRead, FramedWrite};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use open_lambda_proxy_protocol::ProxyMessage;

#[ tokio::main ]
async fn main() {
    let mut argv = std::env::args();
    argv.next().unwrap();

    let container_dir = argv.next().expect("No container directory given");

    simple_logging::log_to_file(format!("{}/db-proxy.log", container_dir), log::LevelFilter::Info).unwrap();

    let path = format!("{}/db-proxy.sock", container_dir);
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
                    Ok((stream, _)) => {
                        tokio::spawn(async move {
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

async fn call_function(func_name: String, arg_string: String) -> Result<String, String> {
    log::debug!("Issuing internal call to {}", func_name);
    let server_addr = "localhost:5000";
    let url = format!("http://{}/run/{}", server_addr, func_name);
    let client = reqwest::Client::new();

    let request = client.post(url).body(arg_string);

    let result = match request.send().await {
        Ok(result) => result,
        Err(err) => {
            return Err(format!("Failed to send call request: {}", err));
        }
    };

    let success = result.status().is_success();

    match result.text().await {
        Ok(result_string) => {
            if success {
                Ok(result_string)
            } else {
                Err(result_string)
            }
        }
        Err(err) => {
            Err(format!("Failed to parse call result: {}", err))
        }
    }
}

async fn handle_connection(stream: UnixStream) {
    log::info!("Connected to process");

    let (reader, writer) = stream.into_split();

    let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
    let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

    log::debug!("Setup database connection");

    while let Some(res) = reader.next().await {
        let data = match res {
            Ok(data) => data,
            Err(e) => {
                log::error!("Failed to receive data from runtime: {}", e);
                break;
            }
        };

        let msg = bincode::deserialize(&data).unwrap();

        let response = match msg {
            ProxyMessage::CallRequest{ func_name, arg_string } => {
                let result = call_function(func_name, arg_string).await;
                ProxyMessage::CallResult(result)
            },
            _ => {
                panic!("Got unexpected message");
            }
        };

        let out_data = bincode::serialize(&response).unwrap();
        writer.send(out_data.into()).await.expect("Failed to send response");
    }
}
