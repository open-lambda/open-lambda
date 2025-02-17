pub use serde_json as json;

use byte_slice_cast::{AsMutByteSlice, ToMutByteSlice};

mod api {
    #[link(wasm_import_module = "ol_args")]
    unsafe extern "C" {
        pub fn get_args(len_out: *mut u64) -> i64;
        pub fn set_result(buf_ptr: *const u8, buf_len: u32);
        pub fn get_unix_time() -> u64;
        pub fn get_random_value(buf_ptr: *mut u8, buf_len: u32);
    }
}

pub fn get_random_value<T: ToMutByteSlice + Default>() -> T {
    let mut value = vec![T::default()];
    unsafe {
        let val_ptr = value.as_mut_byte_slice().as_mut_ptr();
        let val_len = std::mem::size_of::<u64>() as u32;

        api::get_random_value(val_ptr, val_len);
    }
    value.pop().unwrap()
}

pub fn get_unix_time() -> u64 {
    unsafe { api::get_unix_time() }
}

pub fn get_args() -> Option<json::Value> {
    let mut len = 0u64;
    let data_ptr = unsafe {
        let len_ptr = (&mut len) as *mut u64;
        api::get_args(len_ptr)
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

pub fn set_result(value: &json::Value) -> Result<(), json::Error> {
    let val_str = serde_json::to_string(value)?;
    let val_len = val_str.len();

    unsafe { api::set_result(val_str.as_bytes().as_ptr(), val_len as u32) };

    Ok(())
}
