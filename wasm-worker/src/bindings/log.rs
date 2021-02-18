use wasmer::{Array, LazyInit, Function, Memory, Store, Exports, WasmPtr, WasmerEnv};

#[ derive(Clone, Default, WasmerEnv) ]
struct LogEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
}

fn log_info(env: &LogEnv, ptr: WasmPtr<u8, Array>, len: u32) {
    let memory = env.memory.get_ref().unwrap();
    let log_msg = ptr.get_utf8_string(memory, len).unwrap();

    log::info!("Program: {}", log_msg);
}

fn log_debug(env: &LogEnv, ptr: WasmPtr<u8, Array>, len: u32) {
    let memory = env.memory.get_ref().unwrap();
    let log_msg = ptr.get_utf8_string(memory, len).unwrap();

    log::debug!("Program: {}", log_msg);
}

fn log_error(env: &LogEnv, ptr: WasmPtr<u8, Array>, len: u32) {
    let memory = env.memory.get_ref().unwrap();
    let log_msg = ptr.get_utf8_string(memory, len).unwrap();

    log::error!("Program: {}", log_msg);
}
