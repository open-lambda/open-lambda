use open_lambda_proxy_protocol::CallResult;

use serde_bytes::ByteBuf;

mod internal {
    #[link(wasm_import_module = "ol_ipc")]
    extern "C" {
        pub fn host_call(
            ns_name_ptr: *const u8,
            ns_name_len: u32,
            func_name_ptr: *const u8,
            func_name_len: u32,
            arg_data: *const u8,
            arg_data_len: u32,
            result_len_ptr: *mut u64,
        ) -> i64;

        pub fn func_call(
            func_name_ptr: *const u8,
            func_name_len: u32,
            arg_data: *const u8,
            arg_data_len: u32,
            result_len_ptr: *mut u64,
        ) -> i64;
    }
}

pub fn func_call(func_name: &str, args: Vec<u8>) -> CallResult {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::func_call(
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

    Ok(ByteBuf::from(
        unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) }
    ))
}

pub fn host_call(namespace: &str, func_name: &str, args: Vec<u8>) -> CallResult {
    let mut len = 0u64;
    let len_ptr = (&mut len) as *mut u64;

    let data_ptr = unsafe {
        internal::host_call(
            namespace.as_bytes().as_ptr(),
            namespace.len() as u32,
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

    Ok(ByteBuf::from(
        unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) }
    ))
}
