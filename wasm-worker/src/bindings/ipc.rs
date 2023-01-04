use wasmer::{
    Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv, Yielder,
};

use std::net::SocketAddr;

use open_lambda_proxy_protocol::CallResult;

use crate::bindings;
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
        .get_utf8_string(memory, func_name_len).unwrap();

    let args = unsafe {
        let arg_ptr = memory.view::<u8>()
            .as_ptr().add(arg_data_ptr.offset() as usize) as *mut u8;
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
fn host_call(
    env: &IpcEnv,
    ns_name_ptr: WasmPtr<u8, Array>,
    ns_name_len: u32,
    func_name_ptr: WasmPtr<u8, Array>,
    func_name_len: u32,
    arg_data_ptr: WasmPtr<u8, Array>,
    arg_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `function_call` call");

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    // TODO sanity check the namespace name
    let namespace = ns_name_ptr
        .get_utf8_string(memory, ns_name_len).unwrap();

    // TODO sanity check the function name
    let func_name = func_name_ptr
        .get_utf8_string(memory, func_name_len).unwrap();

    let arg_slice = unsafe {
        let arg_ptr = memory.view::<u8>()
            .as_ptr().add(arg_data_ptr.offset() as usize) as *mut u8;
        std::slice::from_raw_parts(arg_ptr, arg_data_len as usize)
    };

    let result: CallResult = yielder.async_suspend(async move {
        if namespace == "lambdastore" {
            cfg_if::cfg_if! {
                if #[ cfg(feature="lambdastore") ] {
                    bindings::extra::lambdastore::call(&func_name, arg_slice).await
                } else {
                    panic!("Feature `lambdastore` not enabled");
                }
            }
        } else {
            panic!("Unknown namespace {namespace}");
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
        "host_call",
        Function::new_native_with_env(store, env.clone(), host_call),
    );

    (ns, env)
}
