pub mod wasm_calls {
    #[link(wasm_import_module = "ol_log")]
    unsafe extern "C" {
        pub fn log_info(msg_ptr: *const u8, msg_len: u32);
        pub fn log_debug(msg_ptr: *const u8, msg_len: u32);
        pub fn log_error(msg_ptr: *const u8, msg_len: u32);
        pub fn log_fatal(msg_ptr: *const u8, msg_len: u32);
    }
}

#[macro_export]
macro_rules! fatal {
    ($($args:tt)*) => {
        let s = format!($($args)*);
        unsafe{
            use $crate::internal::log::wasm_calls;
            wasm_calls::log_fatal(s.as_ptr(), s.len() as u32)
        };

        panic!("Got fatal error; see log.");
    }
}

#[macro_export]
macro_rules! info {
    ($($args:tt)*) => {
        let s = format!($($args)*);
        unsafe{
            use $crate::internal::log::wasm_calls;
            wasm_calls::log_info(s.as_str().as_ptr(), s.len() as u32);
        }
    }
}

#[macro_export]
macro_rules! debug {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{
            use $crate::internal::log::wasm_calls;
            wasm_calls::log_debug(s.as_str().as_ptr(), s.len() as u32);
        }
    }
}

#[macro_export]
macro_rules! error {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{
            use open_lambda::internal::log::wasm_calls;
            wasm_calls::log_error(s.as_str().as_ptr(), s.len() as u32);
        }
    }
}

pub use {debug, error, fatal, info};
