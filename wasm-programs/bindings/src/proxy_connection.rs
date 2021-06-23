use bytes::{Bytes, BytesMut};

use schema::Schema;

use std::io::{Read, Write};

use std::os::unix::net::UnixStream;

use tokio_util::codec::{Encoder, Decoder};
use tokio_util::codec::length_delimited::LengthDelimitedCodec;

use db_proxy_protocol::ProxyMessage;

pub(crate) struct ProxyConnection {
    codec: LengthDelimitedCodec,
    stream: UnixStream,
}

static mut CONNECTION: Option<ProxyConnection> = None;

impl ProxyConnection {
    pub fn get_instance() -> ProxyHandle {
        let inner = if let Some(inner) = unsafe{ CONNECTION.take() } {
            inner
        } else {
            log::debug!("Establishing connection to database proxy");
            let stream = UnixStream::connect("/host/db-proxy.sock").expect("Failed to connect to db-proxy");
            let codec = LengthDelimitedCodec::new();

            Self{ stream, codec }
        };

        ProxyHandle{ inner: Some(inner) }
    }

    pub fn try_get_instance() -> Option<ProxyHandle> {
        unsafe{ CONNECTION.take() }.map(|inner| ProxyHandle{ inner: Some(inner) })
    }

    pub fn get_collection(&mut self, name: String) -> (open_lambda_protocol::CollectionId, Schema) {
        log::debug!("Getting information about collection `{}`", name);

        let msg = ProxyMessage::GetSchema{ collection: name };
        self.send_message(&msg);

        let response = self.receive_message();

        if let ProxyMessage::SchemaResult{ identifier, key, fields } = response {
            let schema = Schema::from_parts(key, fields);
            (identifier, schema)
        } else {
            panic!("Got unexpected response");
        }
    }

    pub fn commit(&mut self) -> bool {
        self.send_message(&ProxyMessage::TxCommitRequest);

        if let ProxyMessage::TxCommitResult{ result } = self.receive_message() {
            result
        } else {
            panic!("got unexpected result");
        }
    }

    pub fn send_message(&mut self, msg: &ProxyMessage) {
        let binmsg = bincode::serialize(msg).unwrap();
        let mut msg_data = BytesMut::new();
        self.codec.encode(Bytes::from(binmsg), &mut msg_data).expect("failed to encode proxy message");
        self.stream.write_all(&msg_data).expect("Failed to send proxy message");
    }

    pub fn receive_message(&mut self) -> ProxyMessage {
        log::trace!("Waiting for message from proxy");

        let mut buffer = BytesMut::new();

        loop {
            let mut data = [0; 1024];
            let len;

            match self.stream.read(&mut data) {
                Ok(l) => { len = l; }
                Err(e) => {
                    panic!("failed to read from socket: {}" , e);
                }
            }

            log::trace!("Received {} bytes from proxy", len);

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
                Err(e) => {
                    panic!("Failed to decode from socket; error = {:?}", e);
                }
            }
        }
    }
}

pub(crate) struct ProxyHandle {
    inner: Option<ProxyConnection>
}

impl ProxyHandle {
    pub fn get_mut(&mut self) -> &mut ProxyConnection {
        self.inner.as_mut().unwrap()
    }
}

impl Drop for ProxyHandle {
    fn drop(&mut self) {
        unsafe {
            CONNECTION = Some(self.inner.take().unwrap());
        }
    }
}
