use open_lambda_proxy_protocol::CallResult;

mod internal {
    #[link(wasm_import_module = "ol_ipc")]
    unsafe extern "C" {
        pub fn http_post(
            address_ptr: *const u8,
            address_len: u32,
            path_ptr: *const u8,
            path_ptr: u32,
            arg_data: *const u8,
            arg_data_len: u32,
            result_len_ptr: *mut u64,
        ) -> i64;

        pub fn http_get(
            address_ptr: *const u8,
            address_len: u32,
            path_ptr: *const u8,
            path_ptr: u32,
            result_len_ptr: *mut u64,
        ) -> i64;

        pub fn function_call(
            func_name_ptr: *const u8,
            func_name_len: u32,
            arg_data: *const u8,
            arg_data_len: u32,
            result_len_ptr: *mut u64,
        ) -> i64;
    }
}

pub fn function_call(func_name: &str, args: Vec<u8>) -> CallResult {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::function_call(
            func_name.as_bytes().as_ptr(),
            func_name.len() as u32,
            args.as_ptr(),
            args.len() as u32,
            len_ptr,
        )
    };

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let call_result_data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };

    bincode::deserialize(&call_result_data).unwrap()
}

pub fn http_get(address: &str, path: &str) -> CallResult {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::http_get(
            address.as_bytes().as_ptr(),
            address.len() as u32,
            path.as_bytes().as_ptr(),
            path.len() as u32,
            len_ptr,
        )
    };

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let call_result_data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };

    bincode::deserialize(&call_result_data).unwrap()
}

pub fn http_post(address: &str, path: &str, args: Vec<u8>) -> CallResult {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::http_post(
            address.as_bytes().as_ptr(),
            address.len() as u32,
            path.as_bytes().as_ptr(),
            path.len() as u32,
            args.as_ptr(),
            args.len() as u32,
            len_ptr,
        )
    };

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let call_result_data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };

    bincode::deserialize(&call_result_data).unwrap()
}
