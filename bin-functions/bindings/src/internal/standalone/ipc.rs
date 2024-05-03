use open_lambda_proxy_protocol::CallResult;

use serde_bytes::ByteBuf;

pub fn function_call<S: ToString>(func_name: S, args: Vec<u8>) -> CallResult {
    let func_name = func_name.to_string();
    log::trace!("Got func_call request for \"{func_name}\"");

    let mut proxy = crate::proxy_connection::ProxyConnection::get_instance();
    proxy.get_mut().func_call(func_name, args)
}

pub fn http_get(address: &str, path: &str) -> CallResult {
    assert!(path.starts_with('/'));

    let url = format!("{address}{path}");
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

    let url = format!("{address}{path}");
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
