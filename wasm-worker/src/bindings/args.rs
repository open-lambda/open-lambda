use wasmer::{Array, Store, WasmerEnv, Exports, Function, Memory, WasmPtr, LazyInit};

use std::sync::{Arc, Mutex};

#[ derive(Clone, WasmerEnv) ]
struct ArgData {
    #[ wasmer(export) ]
    memory: LazyInit<Memory>,

    args: Arc<Vec<u8>>,

    result: Arc<Mutex<Option<Vec<u8>>>>,
}

fn get_args_len(env: &ArgData) -> u32 {
    env.args.len() as u32
}

fn get_args(env: &ArgData, buf_ptr: WasmPtr<u8, Array>, buf_len: u32) -> u32 {
    if env.args.len() > (buf_len as usize) {
        panic!("buffer too small");
    }

    let memory = env.memory.get_ref().unwrap();

    unsafe {
        let buf_ptr = memory.view::<u8>().as_ptr().add( buf_ptr.offset() as usize ) as *mut u8;
        std::ptr::copy(env.args.as_ptr(), buf_ptr, env.args.len());
    }

    env.args.len() as u32
}

fn set_result(env: &ArgData, buf_ptr: WasmPtr<u8, Array>, buf_len: u32) {
    log::debug!("Got result of size {}", buf_len);

    let mut result = env.result.lock().unwrap();

    if result.is_some() {
        panic!("Result was already set");
    }

    let memory = env.memory.get_ref().unwrap();

    let buf_slice = unsafe {
        let buf_ptr = memory.view::<u8>().as_ptr().add( buf_ptr.offset() as usize ) as *mut u8;
        std::slice::from_raw_parts(buf_ptr, buf_len as usize)
    };

    let mut vec = Vec::new();
    vec.extend_from_slice(buf_slice);

    *result = Some(vec);
}

pub fn get_imports(store: &Store, args: Vec<u8>, result: Arc<Mutex<Option<Vec<u8>>>>) -> Exports {
    let arg_data = ArgData{ args: Arc::new(args), result, memory: Default::default() };

    let mut ns = Exports::new();
    ns.insert("ol_set_result", Function::new_native_with_env(&store, arg_data.clone(), set_result));
    ns.insert("ol_get_args_len", Function::new_native_with_env(&store, arg_data.clone(), get_args_len));
    ns.insert("ol_get_args", Function::new_native_with_env(&store, arg_data.clone(), get_args));

    ns
}
