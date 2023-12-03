use open_lambda_proxy_protocol::CallResult;

pub fn function_call<S: ToString>(func_name: S, args: Vec<u8>) -> CallResult {
    let func_name = func_name.to_string();
    log::trace!("Got func_call request for \"{func_name}\"");

    let mut proxy = crate::proxy_connection::ProxyConnection::get_instance();
    proxy.get_mut().func_call(func_name, args)
}

pub fn http_get(address: &str, path: &str) -> CallResult {
    todo!();
}

pub fn http_post(address: &str, path: &str, args: Vec<u8>) -> CallResult {
    todo!();
}
