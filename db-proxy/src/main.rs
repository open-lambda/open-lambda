use std::os::unix::net::UnixListener as StdUnixListener;

use tokio::net::UnixStream;

use futures_util::stream::StreamExt;

use tokio_util::codec::{FramedRead, FramedWrite};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use nix::unistd::{fork, ForkResult};

use lambda_store_client::{create_client, Operation};

fn main() {
    let container_dir = {
        let mut argv = std::env::args();
        argv.next().expect("Expected exactly one argument")
    };

    let connection = {
        let sock = StdUnixListener::bind(format!("{}/host/db-proxy.sock", container_dir))
            .expect("Failed to bind db-proxy unix socket");

        // Fork to let parent know we bound socket successfully
        if let ForkResult::Parent{..} = unsafe{ fork().expect("Fork failed") } {
            std::process::exit(0);
        }

        let (conn, _) = sock.accept().expect("Failed to get connection");
        conn
    };

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_io().build().expect("Failed to start tokio");

    /* FIXME
    let local = tokio::task::LocalSet::new();
    local.block_on(&runtime, async move {
    */ 

    // Process data until the runtime disconnects
    runtime.block_on(async move {
        let stream = UnixStream::from_std(connection).unwrap();
        let (reader, writer) = stream.into_split();

        let mut reader = FramedRead::new(reader, LengthDelimitedCodec::new());
        let mut writer = FramedWrite::new(writer, LengthDelimitedCodec::new());

        let client = create_client("localhost").await;

        while let Some(msg) = reader.next().await {
            let msg = msg.expect("Failed to receive data from runtime");
            let op: Operation = bincode::deserialize(&msg).unwrap();

            //let result = client.execute_operation(op).await;
        };
    });
}
