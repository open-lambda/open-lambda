pub use serde_json as json;

#[ cfg(target_arch="wasm32") ]
mod internal {
    #[link(wasm_import_module="ol_args")]
    extern "C" {
        pub fn get_args_len() -> u32;
        pub fn get_args(buf_ptr: *mut u8, buf_len: u32) -> u32;
        pub fn set_result(buf_ptr: *const u8, buf_len: u32);
    }
}

#[ cfg(not(target_arch="wasm32")) ]
pub fn get_args() -> Option<json::Value> {
    let mut args = std::env::args();

    args.next().unwrap();

    if let Some(arg) = args.next() {
        let jvalue = json::from_str(&arg).expect("Failed to parse JSON");
        Some(jvalue) 
    } else {
        None
    }
}

#[ cfg(target_arch="wasm32") ]
pub fn get_args() -> Option<json::Value> {
    let buf_len = unsafe{ internal::get_args_len() };
    let mut arg_buffer = Vec::new();
    arg_buffer.resize(buf_len as usize, 0);

    let arg_len = unsafe{ internal::get_args(arg_buffer[..].as_mut_ptr(), buf_len) };

    if arg_len == 0 {
        return None;
    }

    assert!(arg_len == buf_len);

    let json_str = String::from_utf8(arg_buffer).expect("Failed to parse argument string; not UTF-8?");
    let jvalue = json::from_str(&json_str).expect("Failed to parse JSON");

    Some(jvalue)
}

#[ cfg(not(target_arch="wasm32")) ]
pub fn set_result(_value: &json::Value) -> Result<(), json::Error> {
    todo!();
}

#[ cfg(target_arch="wasm32") ]
pub fn set_result(value: &json::Value) -> Result<(), json::Error> {
    let val_str = serde_json::to_string(value)?;
    let val_len = val_str.len();

    unsafe{ internal::set_result(val_str.as_bytes().as_ptr(), val_len as u32) };

    Ok(())
}
