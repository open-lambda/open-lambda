mod args;
pub use args::*;

mod log;
pub use crate::log::*;

#[ cfg(target_arch="wasm32") ]
pub fn init() {}

#[ cfg(not(target_arch="wasm32")) ]
pub fn init() {
    env_logger::init();
}
