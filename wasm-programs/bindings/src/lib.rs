#![ feature(vec_into_raw_parts) ]

mod args;
pub use args::*;

mod storage;
pub use storage::*;

mod log;
pub use crate::log::*;

#[ cfg(not(target_arch="wasm32")) ]
mod proxy_connection;

#[ cfg(not(target_arch="wasm32")) ]
pub fn internal_init() {
    env_logger::init();
}

#[ cfg(not(target_arch="wasm32")) ]
pub fn internal_destroy() {
    use std::fs::File;
    use proxy_connection::ProxyConnection;

    let success = if let Some(mut proxy) = ProxyConnection::try_get_instance() {
        proxy.get_mut().commit()
    } else {
        true
    };

    // Create file to tell runtime we succeeded
    if success {
        let f = File::create("/tmp/tx_success").unwrap();
        f.sync_all().unwrap();
    }
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
