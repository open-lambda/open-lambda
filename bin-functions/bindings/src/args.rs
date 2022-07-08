pub use serde_json as json;

#[cfg(target_arch = "wasm32")]
mod internal {
    #[link(wasm_import_module = "ol_args")]
    extern "C" {
        pub fn get_args(len_out: *mut u64) -> i64;
        pub fn set_result(buf_ptr: *const u8, buf_len: u32);
    }
}

#[cfg(not(target_arch = "wasm32"))]
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

#[cfg(target_arch = "wasm32")]
pub fn get_args() -> Option<json::Value> {
    let mut len = 0u64;
    let data_ptr = unsafe {
        let len_ptr = (&mut len) as *mut u64;
        internal::get_args(len_ptr)
    };

    if data_ptr == 0 {
        return None;
    }

    if data_ptr <= 0 {
        panic!("Got unexpected error");
    }

    let len = len as usize;

    let data = unsafe { Vec::<u8>::from_raw_parts(data_ptr as *mut u8, len, len) };

    let json_str = String::from_utf8(data).expect("Failed to parse argument string; not UTF-8?");
    let jvalue = json::from_str(&json_str).expect("Failed to parse JSON");

    Some(jvalue)
}

#[cfg(not(target_arch = "wasm32"))]
pub fn set_result(value: &json::Value) -> Result<(), json::Error> {
    use std::fs::File;
    use std::io::Write;

    let jstr = json::to_string(value)?;

    let path = "/tmp/output";

    let mut file = File::create(path).unwrap();
    file.write_all(jstr.as_bytes()).unwrap();

    file.sync_all().expect("Writing to disk failed");
    log::debug!("Created output file at {}", path);

    Ok(())
}

#[cfg(target_arch = "wasm32")]
pub fn set_result(value: &json::Value) -> Result<(), json::Error> {
    let val_str = serde_json::to_string(value)?;
    let val_len = val_str.len();

    unsafe { internal::set_result(val_str.as_bytes().as_ptr(), val_len as u32) };

    Ok(())
}
