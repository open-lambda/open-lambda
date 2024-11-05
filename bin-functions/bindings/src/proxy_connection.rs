use bytes::{Bytes, BytesMut};

use std::io::{Read, Write};
use std::os::unix::net::UnixStream;

use parking_lot::{Mutex, MutexGuard};
use std::sync::OnceLock;

use tokio_util::codec::length_delimited::LengthDelimitedCodec;
use tokio_util::codec::{Decoder, Encoder};

use serde_bytes::ByteBuf;

use open_lambda_proxy_protocol::{CallResult, FuncCallData, ProxyMessage};

pub(crate) struct ProxyConnection {
    codec: LengthDelimitedCodec,
    stream: UnixStream,
}

pub static CONNECTION: OnceLock<Mutex<ProxyConnection>> = OnceLock::new();

impl ProxyConnection {
    pub fn get() -> MutexGuard<'static, Self> {
        CONNECTION.get_or_init(Self::establish_connection).lock()
    }

    fn establish_connection() -> Mutex<Self> {
        log::debug!("Establishing connection to container proxy");
        let stream =
            UnixStream::connect("/host/proxy.sock").expect("Failed to connect to container proxy");
        let codec = LengthDelimitedCodec::new();

        Mutex::new(Self { stream, codec })
    }

    pub fn func_call(&mut self, fn_name: String, args: Vec<u8>) -> CallResult {
        log::trace!("Issuing call request");
        let cdata = FuncCallData {
            fn_name,
            args: ByteBuf::from(args),
        };
        self.send_message(&ProxyMessage::FuncCallRequest(cdata));

        if let ProxyMessage::FuncCallResult(result) = self.receive_message() {
            result
        } else {
            panic!("got unexpected result");
        }
    }

    pub fn send_message(&mut self, msg: &ProxyMessage) {
        let binmsg = bincode::serialize(msg).unwrap();
        let mut msg_data = BytesMut::new();
        self.codec
            .encode(Bytes::from(binmsg), &mut msg_data)
            .expect("failed to encode proxy message");
        self.stream
            .write_all(&msg_data)
            .expect("Failed to send proxy message");
    }

    pub fn receive_message(&mut self) -> ProxyMessage {
        log::trace!("Waiting for message from proxy");

        let mut buffer = BytesMut::new();

        loop {
            let mut data = [0; 1024];
            let len = match self.stream.read(&mut data) {
                Ok(l) => l,
                Err(err) => panic!("failed to read from socket: {err}"),
            };

            log::trace!("Received {len} bytes from proxy");

            if len > 0 {
                buffer.extend_from_slice(&data[0..len]);
            } else {
                panic!("Lost connection to proxy");
            }

            match self.codec.decode(&mut buffer) {
                Ok(Some(data)) => {
                    return bincode::deserialize(&data).unwrap();
                }
                Ok(None) => {
                    continue;
                }
                Err(err) => panic!("Failed to decode from socket: {err}"),
            }
        }
    }
}
