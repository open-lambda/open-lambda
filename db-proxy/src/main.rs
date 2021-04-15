use tokio::net::{UnixListener, UnixStream};
use tokio::fs::File;
use tokio::io::AsyncWriteExt;

use futures_util::stream::StreamExt;
use futures_util::sink::SinkExt;

use tokio_util::codec::{FramedRead, FramedWrite};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use db_proxy_protocol::ProxyMessage;

use lambda_store_client::create_client;

fn main() {
    let container_dir = {
        let mut argv = std::env::args();
        argv.next().unwrap();
        argv.next().expect("Expected exactly one argument")
    };

    simple_logging::log_to_file(format!("{}/db-proxy.log", container_dir), log::LevelFilter::Debug).unwrap();

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(4)
        .enable_io().build().expect("Failed to start tokio");

    // Process data until the runtime disconnects
    runtime.block_on(async move {
        let path = format!("{}/db-proxy.sock", container_dir);
        let listener = UnixListener::bind(path).unwrap();

        // Create pid file to notify others that the socket is bound
        let mut file = File::create(format!("{}/db-proxy.pid", container_dir)).await.unwrap();

        let content = format!("{}", std::process::id()).into_bytes();
        file.write_all(&content).await.unwrap();

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
                        Err(e) => {
                            log::error!("Got error: {}", e);
                            std::process::exit(-1);
                        }
                    }
                }
            }
        }
    });

    log::info!("Shutting down database proxy");
}

async fn handle_connection(stream: UnixStream) {
    log::info!("Connected to process");
    stream.set_nodelay(true);

    let (reader, writer) = stream.into_split();

    let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
    let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

    let client = create_client("localhost").await;

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
            ProxyMessage::GetSchema{ collection: col_name } => {
                let col = client.get_collection(col_name).expect("No such collection");
                let (key, fields) = col.get_schema().clone_inner();
                let identifier = col.get_identifier();

                ProxyMessage::SchemaResult{ identifier, key, fields }
            }
            ProxyMessage::ExecuteOperation{ collection, op } => {
                let ntype = if op.is_write() {
                    lambda_store_client::NodeType::Head
                } else {
                    lambda_store_client::NodeType::Tail
                };

                let collection = client.get_collection_by_id(collection).unwrap();
                let result = collection.execute_operation(op, ntype).await;

                ProxyMessage::OperationResult{ result }
            }
            _ => {
                panic!("Got unexpected message");
            }
        };

        let out_data = bincode::serialize(&response).unwrap();
        writer.send(out_data.into()).await.expect("Failed to send response");
    }

    client.close().await;
}
