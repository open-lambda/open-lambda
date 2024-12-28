use open_lambda_proxy_protocol::CallResult;

use serde_bytes::ByteBuf;

use crate::proxy_connection::ProxyConnection;

pub fn function_call<S: ToString>(func_name: S, args: Vec<u8>) -> CallResult {
    let func_name = func_name.to_string();
    log::trace!("Got func_call request for \"{func_name}\"");

    ProxyConnection::get().func_call(func_name, args)
}

pub fn http_get(address: &str, path: &str) -> CallResult {
    assert!(path.starts_with('/'));

    let url = format!("http://{address}{path}");
    let mut result = vec![];

    match ureq::get(&url)
        .call()
        .map_err(|err| {
            format!("Failed to send request to {url}: {err}")
        })?
        .into_reader()
        .read_to_end(&mut result)
    {
        Ok(_) => Ok(ByteBuf::from(result)),
        Err(err) => Err(err.to_string()),
    }
}

pub fn http_post(address: &str, path: &str, args: Vec<u8>) -> CallResult {
    assert!(path.starts_with('/'));

    let url = format!("http://{address}{path}");
    let mut result = vec![];

    match ureq::post(&url)
        .send_bytes(&args)
        .map_err(|err| {
            format!("Failed to send request to {url}: {err}")
        })?
        .into_reader()
        .read_to_end(&mut result)
    {
        Ok(_) => Ok(ByteBuf::from(result)),
        Err(err) => Err(err.to_string()),
    }
}
