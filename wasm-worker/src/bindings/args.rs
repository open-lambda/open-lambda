use wasmer::{Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv};

use std::sync::{Arc, Mutex};

#[derive(Clone, WasmerEnv)]
struct ArgData {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    args: Arc<Vec<u8>>,
    result: Arc<Mutex<Option<Vec<u8>>>>,
}

fn get_args(env: &ArgData, len_out: WasmPtr<u64>) -> i64 {
    log::trace!("Got `get_args` call");

    let memory = env.memory.get_ref().unwrap();

    let args = &env.args;

    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(args.len() as u32)
        .unwrap();

    if args.len() == 0 {
        return 0;
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, args.len())
    };

    out_slice.clone_from_slice(args.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(args.len() as u64);

    offset
}

fn set_result(env: &ArgData, buf_ptr: WasmPtr<u8, Array>, buf_len: u32) {
    log::debug!("Got result of size {}", buf_len);

    let mut result = env.result.lock().unwrap();

    if result.is_some() {
        panic!("Result was already set");
    }

    let memory = env.memory.get_ref().unwrap();

    let buf_slice = unsafe {
        let buf_ptr = memory.view::<u8>().as_ptr().add(buf_ptr.offset() as usize) as *mut u8;
        std::slice::from_raw_parts(buf_ptr, buf_len as usize)
    };

    let mut vec = Vec::new();
    vec.extend_from_slice(buf_slice);

    *result = Some(vec);
}

fn get_random_value() -> u64 {
    rand::random()
}

pub fn get_imports(
    store: &Store,
    args: Arc<Vec<u8>>,
    result: Arc<Mutex<Option<Vec<u8>>>>,
) -> Exports {
    let arg_data = ArgData {
        args,
        result,
        memory: Default::default(),
        allocate: Default::default(),
    };

    let mut ns = Exports::new();
    ns.insert(
        "set_result",
        Function::new_native_with_env(store, arg_data.clone(), set_result),
    );
    ns.insert(
        "get_args",
        Function::new_native_with_env(store, arg_data, get_args),
    );
    ns.insert(
        "get_random_value",
        Function::new_native(store, get_random_value),
    );

    ns
}
