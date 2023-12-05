use std::collections::HashMap;

use serde_bytes::ByteBuf;

use wasmer::{Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv};

use open_lambda_proxy_protocol::CallResult;

#[derive(Clone, WasmerEnv)]
pub struct ConfigEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,

    config_values: HashMap<String, String>,
}

fn get_config_value(
    env: &ConfigEnv,
    key_ptr: WasmPtr<u8, Array>,
    key_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    let memory = env.memory.get_ref().unwrap();

    let key = key_ptr.get_utf8_string(memory, key_len).unwrap();

    let result: CallResult = match env.config_values.get(&key) {
        Some(val) => Ok(ByteBuf::from(bincode::serialize(val).unwrap())),
        None => Err("No such config value".to_string()),
    };

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

pub fn get_imports(store: &Store, config_values: HashMap<String, String>) -> (Exports, ConfigEnv) {
    let mut ns = Exports::new();
    let env = ConfigEnv {
        memory: Default::default(),
        allocate: Default::default(),
        config_values,
    };

    ns.insert(
        "get_config_value",
        Function::new_native_with_env(store, env.clone(), get_config_value),
    );

    (ns, env)
}
