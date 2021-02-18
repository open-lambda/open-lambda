#[link(wasm_import_module="ol_log")]
extern "C" {
    pub fn ol_log_info(msg_ptr: *const u8, msg_len: u32);
    pub fn ol_log_debug(msg_ptr: *const u8, msg_len: u32);
    pub fn ol_log_error(msg_ptr: *const u8, msg_len: u32);
}

#[macro_export]
macro_rules! info {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ open_lambda::ol_log_info(s.as_str().as_ptr(), s.len() as u32); }
    }
}

#[macro_export]
macro_rules! debug {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ opend_lambda::ol_log_debug(s.as_str().as_ptr(), s.len() as u32); }
    }
}

#[macro_export]
macro_rules! error {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ open_lambda::ol_log_error(s.as_str().as_ptr(), s.len() as u32); }
    }
}
