use open_lambda_proxy_protocol::CallResult;

pub fn function_call<S: ToString>(func_name: S, args: Vec<u8>) -> CallResult {
    let func_name = func_name.to_string();
    log::trace!("Got func_call request for \"{func_name}\"");

    let mut proxy = crate::proxy_connection::ProxyConnection::get_instance();
    proxy.get_mut().func_call(func_name, args)
}

pub fn host_call<S: ToString>(namespace: S, func_name: S, args: Vec<u8>) -> CallResult {
    let namespace = namespace.to_string();
    let func_name = func_name.to_string();
    log::trace!("Got host_call request for \"{namespace}::{func_name}\"");

    let mut proxy = crate::proxy_connection::ProxyConnection::get_instance();
    proxy.get_mut().host_call(namespace, func_name, args)
}
