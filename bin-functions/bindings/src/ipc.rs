use serde_json as json;

use open_lambda_proxy_protocol::CallResult;

#[cfg(target_arch = "wasm32")]
mod internal {
    #[link(wasm_import_module = "ol_ipc")]
    extern "C" {
        pub fn call(
            func_name_ptr: *const u8,
            func_name_len: u32,
            arg_data: *const u8,
            arg_data_len: u32,
            result_len_ptr: *mut u64,
        ) -> i64;
    }
}

#[cfg(target_arch = "wasm32")]
pub fn call(func_name: &str, args: &json::Value) -> Result<Option<json::Value>, String> {
    let args_str = serde_json::to_string(args).unwrap();
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::call(
            func_name.as_bytes().as_ptr(),
            func_name.len() as u32,
            args_str.as_bytes().as_ptr(),
            args_str.len() as u32,
            len_ptr,
        )
    };

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let result_data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };

    //TODO get rid of this additional serialization
    parse_json_result(bincode::deserialize(&result_data).unwrap())
}

#[cfg(not(target_arch = "wasm32"))]
pub fn call<S: ToString>(func_name: S, args: &json::Value) -> Result<Option<json::Value>, String> {
    let func_name = func_name.to_string();
    log::debug!("Got call request for \"{func_name}\"");

    let arg_string = serde_json::to_string(args).unwrap();
    let jdata = arg_string.as_bytes().to_vec();

    let mut proxy = crate::proxy_connection::ProxyConnection::get_instance();
    let result = proxy.get_mut().call(func_name, jdata);

    parse_json_result(result)
}

fn parse_json_result(result: CallResult) -> Result<Option<json::Value>, String> {
    match result {
        Ok(result_data) => {
            if result_data.is_empty() {
                // Is this needed? Result should always be a valid json
                Ok(None)
            } else {
                let result_string = String::from_utf8(result_data.into_vec())
                    .expect("Result not a valid utf-8 string?");

                match json::from_str(&result_string) {
                    Ok(json) => Ok(Some(json)),
                    Err(err) => Err(format!(
                        "Failed to parse call result JSON \"{result_string}\": {err}"
                    )),
                }
            }
        }
        Err(msg) => Err(msg),
    }
}
