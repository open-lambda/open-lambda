use open_lambda_proxy_protocol::CallResult;

mod api {
    #[link(wasm_import_module = "ol_config")]
    unsafe extern "C" {
        pub fn get_config_value(key_ptr: *const u8, key_len: u32, len_out: *mut u64) -> i64;
    }
}

pub fn get_config_value(key: &str) -> Result<String, String> {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr =
        unsafe { api::get_config_value(key.as_bytes().as_ptr(), key.len() as u32, len_ptr) };

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let result_data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };
    let result: CallResult = bincode::deserialize(&result_data).unwrap();
    result.map(|value| bincode::deserialize(&value).unwrap())
}
