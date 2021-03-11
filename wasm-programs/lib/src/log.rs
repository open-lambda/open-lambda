#[ cfg(target_arch="wasm32") ]
pub mod internal {
    #[link(wasm_import_module="ol_log")]
    extern "C" {
        pub fn log_info(msg_ptr: *const u8, msg_len: u32);
        pub fn log_debug(msg_ptr: *const u8, msg_len: u32);
        pub fn log_error(msg_ptr: *const u8, msg_len: u32);
    }
}

#[ cfg(target_arch="wasm32") ]
#[macro_export]
macro_rules! info {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ open_lambda::internal::log_info(s.as_str().as_ptr(), s.len() as u32); }
    }
}

#[ cfg(target_arch="wasm32") ]
#[macro_export]
macro_rules! debug {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ open_lambda::internal::log_debug(s.as_str().as_ptr(), s.len() as u32); }
    }
}

#[ cfg(target_arch="wasm32") ]
#[macro_export]
macro_rules! error {
    ($($args:tt)*) => {
        let s = std::format!("{}", std::format_args!($($args)*) );
        unsafe{ open_lambda::internal::log_error(s.as_str().as_ptr(), s.len() as u32); }
    }
}

#[ cfg(not(target_arch="wasm32")) ]
pub use log::{info, debug, error};
