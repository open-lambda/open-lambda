use std::collections::HashMap;
use std::future::Future;

use serde_bytes::ByteBuf;

use open_lambda_proxy_protocol::CallResult;

use super::{BindingsData, call_allocate, fill_slice, get_str, set_u64};

use wasmtime::{Caller, Linker};

pub struct ConfigData {
    config_values: HashMap<String, String>,
}

impl ConfigData {
    pub fn new(config_values: HashMap<String, String>) -> Self {
        Self { config_values }
    }
}

fn get_config_value(
    mut caller: Caller<BindingsData>,
    args: (i32, u32, i32),
) -> Box<dyn Future<Output = i64> + Send + '_> {
    let (key_ptr, key_len, len_out) = args;

    Box::new(async move {
        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        let key = get_str(&caller, &memory, key_ptr, key_len).to_string();

        let result: CallResult = match caller.data().config.config_values.get(&key) {
            Some(val) => Ok(ByteBuf::from(bincode::serialize(val).unwrap())),
            None => Err("No such config value".to_string()),
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
    let module = "ol_config";

    linker
        .func_wrap_async(module, "get_config_value", get_config_value)
        .unwrap();
}
