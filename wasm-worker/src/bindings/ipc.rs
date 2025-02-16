use std::future::Future;
use std::net::SocketAddr;

use serde_bytes::ByteBuf;

use open_lambda_proxy_protocol::CallResult;

use super::{BindingsData, call_allocate, fill_slice, get_slice, get_str, set_u64};

use wasmtime::{Caller, Linker};

use crate::http_client::HttpClient;

#[derive(Clone)]
pub struct IpcData {
    addr: SocketAddr,
}

impl IpcData {
    pub fn new(addr: SocketAddr) -> Self {
        Self { addr }
    }
}

fn function_call(
    mut caller: Caller<'_, BindingsData>,
    args: (i32, u32, i32, u32, i32),
) -> Box<dyn Future<Output = i64> + Send + '_> {
    let (func_name_ptr, func_name_len, arg_data_ptr, arg_data_len, len_out) = args;

    Box::new(async move {
        log::trace!("Got `function_call` call");

        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        let func_name = get_str(&caller, &memory, func_name_ptr, func_name_len);

        let args = get_slice(&caller, &memory, arg_data_ptr, arg_data_len);

        let mut client = HttpClient::new(caller.data().ipc.addr).await;

        let response = match client
            .post(format!("/run/{func_name}"), args.to_vec())
            .await
        {
            Ok(resp) => resp,
            Err(err) => {
                panic!("Internal call to {} failed: {err}", caller.data().ipc.addr);
            }
        };

        let result: CallResult = Ok(ByteBuf::from(response));

        let result_data = bincode::serialize(&result).unwrap();
        let buffer_len = result_data.len();

        let offset = call_allocate(&mut caller, buffer_len as u32).await;

        fill_slice(&caller, &memory, offset, result_data.as_slice());
        set_u64(&caller, &memory, len_out, buffer_len as u64);

        offset as i64
    })
}

#[allow(clippy::too_many_arguments)]
fn http_post(
    mut caller: Caller<'_, BindingsData>,
    args: (i32, u32, i32, u32, i32, u32, i32),
) -> Box<dyn Future<Output = i64> + Send + '_> {
    let (addr_ptr, addr_len, path_ptr, path_len, body_data_ptr, body_data_len, len_out) = args;
    Box::new(async move {
        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        let addr = get_str(&caller, &memory, addr_ptr, addr_len);
        let path = get_str(&caller, &memory, path_ptr, path_len);

        log::trace!("Got `http_post` call to {addr} with path={path}");

        let body_slice = get_slice(&caller, &memory, body_data_ptr, body_data_len);

        let mut client = HttpClient::new(addr).await;
        let result: CallResult = match client.post(path.to_string(), body_slice.to_vec()).await {
            Ok(data) => Ok(ByteBuf::from(data)),
            Err(err) => Err(err.to_string()),
        };

        let result_data = bincode::serialize(&result).unwrap();
        let buffer_len = result_data.len();

        let offset = call_allocate(&mut caller, buffer_len as u32).await;

        fill_slice(&caller, &memory, offset, result_data.as_slice());
        set_u64(&caller, &memory, len_out, buffer_len as u64);

        offset as i64
    })
}

#[allow(clippy::too_many_arguments)]
fn http_get(
    mut caller: Caller<'_, BindingsData>,
    args: (i32, u32, i32, u32, i32),
) -> Box<dyn Future<Output = i64> + Send + '_> {
    let (addr_ptr, addr_len, path_ptr, path_len, len_out) = args;

    Box::new(async move {
        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        let addr = get_str(&caller, &memory, addr_ptr, addr_len);
        let path = get_str(&caller, &memory, path_ptr, path_len);

        log::trace!("Got `http_post` call to {addr} with path={path}");

        let mut client = HttpClient::new(addr).await;
        let result: CallResult = match client.get(path.to_string()).await {
            Ok(data) => Ok(ByteBuf::from(data)),
            Err(err) => Err(err.to_string()),
        };

        let result_data = bincode::serialize(&result).unwrap();
        let buffer_len = result_data.len();

        let offset = call_allocate(&mut caller, buffer_len as u32).await;

        fill_slice(&caller, &memory, offset, result_data.as_slice());
        set_u64(&caller, &memory, len_out, buffer_len as u64);

        offset as i64
    })
}

pub fn get_imports(linker: &mut Linker<BindingsData>) {
    linker
        .func_wrap_async("ol_ipc", "function_call", function_call)
        .unwrap();
    linker
        .func_wrap_async("ol_ipc", "http_post", http_post)
        .unwrap();
    linker
        .func_wrap_async("ol_ipc", "http_get", http_get)
        .unwrap();
}
