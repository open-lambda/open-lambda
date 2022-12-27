#[cfg(not(target_arch = "wasm32"))]
mod standalone;

#[cfg(not(target_arch = "wasm32"))]
pub use standalone::*;

#[cfg(target_arch = "wasm32")]
mod wasm;

#[cfg(target_arch = "wasm32")]
pub use wasm::*;
