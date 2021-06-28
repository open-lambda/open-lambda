use tokio::net::{UnixListener, UnixStream};
use tokio::fs::File;
use tokio::io::AsyncWriteExt;

use futures_util::stream::StreamExt;
use futures_util::sink::SinkExt;

use tokio_util::codec::{FramedRead, FramedWrite};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use db_proxy_protocol::ProxyMessage;

use lambda_store_client::create_client;

#[ tokio::main ]
async fn main() {
    let container_dir = {
        let mut argv = std::env::args();
        argv.next().unwrap();
        argv.next().expect("Expected exactly one argument")
    };

    simple_logging::log_to_file(format!("{}/db-proxy.log", container_dir), log::LevelFilter::Debug).unwrap();

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

    let result_string = match result.text().await {
        Ok(result_string) => result_string,
        Err(err) => {
            return Err(format!("Failed to parse call result: {}", err));
        }
    };

    Ok(result_string)
}

async fn handle_connection(stream: UnixStream) {
    log::info!("Connected to process");

    let (reader, writer) = stream.into_split();

    let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
    let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

    // FIXME: reuse or create lazily
    let client = create_client("localhost").await;
    let mut tx = Some(client.begin_transaction());
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

                let tx = tx.as_ref().expect("Transaction already committed?");
                let collection = tx.get_collection_by_id(collection).unwrap();
                let result = collection.execute_operation(op, ntype).await;

                ProxyMessage::OperationResult{ result }
            }
            ProxyMessage::TxCommitRequest => {
                let tx = tx.take().expect("Transaction already committed?");
                let result = tx.commit().await.is_ok();

                ProxyMessage::TxCommitResult{result}
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
