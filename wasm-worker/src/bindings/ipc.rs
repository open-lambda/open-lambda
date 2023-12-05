use wasmer::{
    Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv, Yielder,
};

use std::net::SocketAddr;

use serde_bytes::ByteBuf;

use open_lambda_proxy_protocol::CallResult;

use crate::http_client::HttpClient;

#[derive(Clone, WasmerEnv)]
pub struct IpcEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
    addr: SocketAddr,
}

fn function_call(
    env: &IpcEnv,
    func_name_ptr: WasmPtr<u8, Array>,
    func_name_len: u32,
    arg_data_ptr: WasmPtr<u8, Array>,
    arg_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `function_call` call");

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    // TODO sanity check the function name
    let func_name = func_name_ptr
        .get_utf8_string(memory, func_name_len)
        .unwrap();

    let args = unsafe {
        let arg_ptr = memory
            .view::<u8>()
            .as_ptr()
            .add(arg_data_ptr.offset() as usize) as *mut u8;
        let slice = std::slice::from_raw_parts(arg_ptr, arg_data_len as usize);
        let mut args = vec![];
        args.extend_from_slice(slice);
        args
    };

    let result: CallResult = yielder.async_suspend(async move {
        let mut client = HttpClient::new(env.addr).await;

        let response = match client.post(format!("/run/{func_name}"), args).await {
            Ok(resp) => resp,
            Err(err) => {
                panic!("Internal call to {} failed: {err}", env.addr);
            }
        };

        Ok(bincode::deserialize(&response).unwrap())
    });

    let result_data = bincode::serialize(&result).unwrap();
    let buffer_len = result_data.len();
    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(buffer_len as u32)
        .unwrap();

    if offset < 0 {
        panic!("Failed to allocate");
    }

    if (offset as u64) + (buffer_len as u64) > memory.data_size() {
        panic!("Invalid pointer");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, buffer_len)
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(buffer_len as u64);

    offset
}

#[allow(clippy::too_many_arguments)]
fn http_post(
    env: &IpcEnv,
    addr_ptr: WasmPtr<u8, Array>,
    addr_len: u32,
    path_ptr: WasmPtr<u8, Array>,
    path_len: u32,
    body_data_ptr: WasmPtr<u8, Array>,
    body_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `function_call` call");

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    let addr = addr_ptr.get_utf8_string(memory, addr_len).unwrap();
    let path = path_ptr.get_utf8_string(memory, path_len).unwrap();

    let body_slice = unsafe {
        let arg_ptr = memory
            .view::<u8>()
            .as_ptr()
            .add(body_data_ptr.offset() as usize) as *mut u8;
        std::slice::from_raw_parts(arg_ptr, body_data_len as usize)
    };

    let result: CallResult = yielder.async_suspend(async move {
        let mut client = HttpClient::new(addr).await;
        match client.post(path, body_slice.to_vec()).await {
            Ok(data) => Ok(ByteBuf::from(data)),
            Err(err) => Err(err.to_string()),
        }
    });

    let result_data = bincode::serialize(&result).unwrap();
    let buffer_len = result_data.len();
    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(buffer_len as u32)
        .unwrap();

    if offset < 0 {
        panic!("Failed to allocate");
    }

    if (offset as u64) + (buffer_len as u64) > memory.data_size() {
        panic!("Invalid pointer");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, buffer_len)
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(buffer_len as u64);

    offset
}

#[allow(clippy::too_many_arguments)]
fn http_get(
    env: &IpcEnv,
    addr_ptr: WasmPtr<u8, Array>,
    addr_len: u32,
    path_ptr: WasmPtr<u8, Array>,
    path_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `function_call` call");

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    let addr = addr_ptr.get_utf8_string(memory, addr_len).unwrap();
    let path = path_ptr.get_utf8_string(memory, path_len).unwrap();

    let result: CallResult = yielder.async_suspend(async move {
        let mut client = HttpClient::new(addr).await;
        match client.get(path).await {
            Ok(data) => Ok(ByteBuf::from(data)),
            Err(err) => Err(err.to_string()),
        }
    });

    let result_data = bincode::serialize(&result).unwrap();
    let buffer_len = result_data.len();
    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(buffer_len as u32)
        .unwrap();

    if offset < 0 {
        panic!("Failed to allocate");
    }

    if (offset as u64) + (buffer_len as u64) > memory.data_size() {
        panic!("Invalid pointer");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, buffer_len)
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(buffer_len as u64);

    offset
}

pub fn get_imports(store: &Store, addr: SocketAddr) -> (Exports, IpcEnv) {
    let mut ns = Exports::new();
    let env = IpcEnv {
        memory: Default::default(),
        allocate: Default::default(),
        yielder: Default::default(),
        addr,
    };

    ns.insert(
        "function_call",
        Function::new_native_with_env(store, env.clone(), function_call),
    );

    ns.insert(
        "http_post",
        Function::new_native_with_env(store, env.clone(), http_post),
    );

    ns.insert(
        "http_get",
        Function::new_native_with_env(store, env.clone(), http_get),
    );

    (ns, env)
}
