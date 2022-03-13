use wasmer::{Array, Exports, Function, LazyInit, Memory, Store, WasmPtr, WasmerEnv};

#[derive(Clone, Default, WasmerEnv)]
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

fn log_fatal(env: &LogEnv, ptr: WasmPtr<u8, Array>, len: u32) {
    let memory = env.memory.get_ref().unwrap();
    let log_msg = ptr.get_utf8_string(memory, len).unwrap();

    log::error!("Program has fatal error: {}", log_msg);
}

pub fn get_imports(store: &Store) -> Exports {
    let mut ns = Exports::new();
    ns.insert(
        "log_info",
        Function::new_native_with_env(store, LogEnv::default(), log_info),
    );
    ns.insert(
        "log_debug",
        Function::new_native_with_env(store, LogEnv::default(), log_debug),
    );
    ns.insert(
        "log_error",
        Function::new_native_with_env(store, LogEnv::default(), log_error),
    );
    ns.insert(
        "log_fatal",
        Function::new_native_with_env(store, LogEnv::default(), log_fatal),
    );

    ns
}
