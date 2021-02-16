pub use serde_json as json;

#[link(wasm_import_module="open_lambda")]
extern "C" {
    fn ol_get_args(buf_ptr: *mut u8, buf_len: u32) -> u32;
    fn ol_set_result(buf_ptr: *const u8, buf_len: u32);
}

pub fn get_args() -> Option<json::Value> {
    const BUF_LEN: usize = 1024;
    let mut arg_buffer = [0u8; BUF_LEN];

    let arg_len = unsafe{ ol_get_args(arg_buffer[..].as_mut_ptr(), BUF_LEN as u32) };

    if arg_len == 0 {
        return None;
    }

    let mut data = Vec::new();
    data.extend_from_slice(&arg_buffer[0..(arg_len as usize)]);

    let json_str = String::from_utf8(data).expect("Failed to parse argument string; not UTF-8?");
    let jvalue = json::from_str(&json_str).expect("Failed to parse JSON");

    Some(jvalue)
}

pub fn set_result(value: json::Value) {
    let val_str = value.as_str().unwrap();
    let val_len = val_str.len();

    unsafe{ ol_set_result(val_str.as_bytes().as_ptr(), val_len as u32) };
}
