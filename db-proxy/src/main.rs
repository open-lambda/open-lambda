use std::os::unix::net::UnixListener as StdUnixListener;

use tokio::net::UnixListener;

use futures_util::stream::StreamExt;
use futures_util::sink::SinkExt;

use tokio_util::codec::{FramedRead, FramedWrite};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use nix::unistd::{fork, ForkResult};

use db_proxy_protocol::ProxyMessage;

use lambda_store_client::create_client;

fn main() {
    let container_dir = {
        let mut argv = std::env::args();
        argv.next().unwrap();
        argv.next().expect("Expected exactly one argument")
    };

    simple_logging::log_to_file(format!("{}/db-proxy.log", container_dir), log::LevelFilter::Debug).unwrap();

    let sock = {
        let path = format!("{}/db-proxy.sock", container_dir);
        let sock = match StdUnixListener::bind(&path) {
            Ok(sock) => {
                log::info!("Bound socket at `{}`", path);
                sock
            }
            Err(e) => {
                panic!("Failed to bind db-proxy unix socket at `{}`: {}", path, e);
            }
        };

        // Fork to let parent know we bound socket successfully
        if let ForkResult::Parent{..} = unsafe{ fork().expect("Fork failed") } {
            std::process::exit(0);
        }

        sock
    };

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(4)
        .enable_io().build().expect("Failed to start tokio");

    // Process data until the runtime disconnects
    runtime.block_on(async move {
        let stream = {
            let listener = UnixListener::from_std(sock).unwrap();
            let (stream, _) = listener.accept().await.expect("Failed to get connection");

            stream
        };

        log::info!("Connected to process");
        let (reader, writer) = stream.into_split();

        let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
        let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

        let client = create_client("localhost").await;

        while let Some(data) = reader.next().await {
            let data = data.expect("Failed to receive data from runtime");
            let msg = bincode::deserialize(&data).unwrap();

            let response = match msg {
                ProxyMessage::GetSchema{ collection: col_name } => {
                    let col = client.get_collection(col_name).expect("No such collection");
                    let (key, fields) = col.get_schema().clone_inner();
                    let identifier = col.get_identifier();

                    ProxyMessage::SchemaResult{ identifier, key, fields }
                }
                ProxyMessage::ExecuteOperation{ collection, op } => {
                    let collection = client.get_collection_by_id(collection).unwrap();
                    let result = collection.execute_operation(op).await;

                    ProxyMessage::OperationResult{ result }
                }
                _ => {
                    panic!("Got unexpected message");
                }
            };

            let out_data = bincode::serialize(&response).unwrap();
            writer.send(out_data.into()).await.expect("Failed to send response");
        };
    });
}
