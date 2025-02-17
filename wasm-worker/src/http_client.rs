use tokio::net::ToSocketAddrs;

use hyper::Request;
use hyper::body::Bytes;
use hyper::client::conn;

use crate::support;

use http_body_util::{BodyExt, Full};

pub struct HttpClient {
    request_sender: conn::http1::SendRequest<Full<Bytes>>,
}

impl HttpClient {
    pub async fn new<S: ToSocketAddrs + std::fmt::Debug>(server_addr: S) -> Self {
        let conn = tokio::net::TcpStream::connect(&server_addr)
            .await
            .unwrap_or_else(|err| {
                panic!("Failed to connect to HTTP server at {server_addr:?}: {err}")
            });
        conn.set_nodelay(true).unwrap();
        let conn = support::TokioIo::new(conn);

        let (request_sender, connection) = conn::http1::handshake(conn)
            .await
            .expect("HTTP handshake failed");

        tokio::spawn(async move {
            if let Err(err) = connection.await {
                log::error!("Got HTTP error: {err}");
            }
        });

        Self { request_sender }
    }

    pub async fn get(&mut self, path: String) -> Result<Vec<u8>, hyper::Error> {
        let request = Request::builder()
            .method("GET")
            .uri(path)
            .body(Full::new(Bytes::from("")))
            .unwrap();

        let response = self.request_sender.send_request(request).await?;

        Ok(response.collect().await?.to_bytes().to_vec())
    }

    pub async fn post(&mut self, path: String, content: Vec<u8>) -> Result<Vec<u8>, hyper::Error> {
        let request = Request::builder()
            .method("POST")
            .uri(path)
            .body(Full::new(Bytes::from(content)))
            .unwrap();

        let response = self.request_sender.send_request(request).await?;
        Ok(response.collect().await?.to_bytes().to_vec())
    }
}
