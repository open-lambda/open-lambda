use wasmtime::{Caller, Linker};

use super::{BindingsData, get_str};

fn log_info(mut caller: Caller<'_, BindingsData>, ptr: i32, len: u32) {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let log_msg = get_str(&caller, &memory, ptr, len);
    log::info!("Program: {log_msg}");
}

fn log_debug(mut caller: Caller<'_, BindingsData>, ptr: i32, len: u32) {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let log_msg = get_str(&caller, &memory, ptr, len);
    log::debug!("Program: {log_msg}");
}

fn log_error(mut caller: Caller<'_, BindingsData>, ptr: i32, len: u32) {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let log_msg = get_str(&caller, &memory, ptr, len);
    log::error!("Program has error: {log_msg}");
}

fn log_fatal(mut caller: Caller<'_, BindingsData>, ptr: i32, len: u32) {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let log_msg = get_str(&caller, &memory, ptr, len);
    log::error!("Program has fatal error: {log_msg}");
}

pub fn get_imports(linker: &mut Linker<BindingsData>) {
    let module = "ol_log";

    linker.func_wrap(module, "log_info", log_info).unwrap();
    linker.func_wrap(module, "log_debug", log_debug).unwrap();
    linker.func_wrap(module, "log_error", log_error).unwrap();
    linker.func_wrap(module, "log_fatal", log_fatal).unwrap();
}
