#![ feature(vec_into_raw_parts) ]

mod args;
pub use args::*;

mod storage;
pub use storage::*;

mod log;
pub use crate::log::*;

#[ cfg(not(target_arch="wasm32")) ]
mod proxy_connection;

#[ cfg(target_arch="wasm32") ]
pub fn init() {}

#[ cfg(not(target_arch="wasm32")) ]
pub fn init() {
    env_logger::init();

    //Initialize connection so proxy does not run forever
 //   proxy_connection::ProxyConnection::get_instance();
}

#[ cfg(target_arch="wasm32") ]
#[ no_mangle ]
fn internal_alloc_buffer(size: u32) -> i64 {
    let size = size as usize;

    let mut vec = Vec::<u8>::new();
    vec.reserve(size);

    let (ptr, _, vec_len) = vec.into_raw_parts();
    assert!(vec_len == size);

    ptr as i64
}
