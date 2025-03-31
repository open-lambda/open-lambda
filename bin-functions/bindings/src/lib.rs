#![feature(vec_into_raw_parts)]
#![feature(once_cell_get_mut)]

use rand::distr::{Distribution, StandardUniform};
use rand::rngs::SmallRng;
use rand::{Rng, SeedableRng};

mod args;
pub use args::*;

mod config;
pub use config::*;

mod ipc;
pub use crate::ipc::*;

pub mod log;

pub mod internal;

#[cfg(not(target_arch = "wasm32"))]
mod proxy_connection;

#[cfg(not(target_arch = "wasm32"))]
pub fn internal_init() {
    env_logger::init();
}

#[inline]
pub fn rng() -> SmallRng {
    SmallRng::from_os_rng()
}

#[inline]
pub fn random<T>() -> T
where
    StandardUniform: Distribution<T>,
{
    rng().random()
}

#[cfg(target_arch = "wasm32")]
#[unsafe(no_mangle)]
fn internal_alloc_buffer(size: u32) -> i64 {
    let size = size as usize;
    let vec = Vec::<u8>::with_capacity(size);

    let (ptr, _, vec_len) = vec.into_raw_parts();
    assert_eq!(vec_len, size);

    ptr as i64
}

pub fn set_panic_handler() {
    std::panic::set_hook(Box::new(|err| {
        crate::log::fatal!("Got panic: {err}");
    }));
}
