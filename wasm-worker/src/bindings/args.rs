use wasmer::{Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv};

use std::sync::Arc;

use rand::Fill;

use parking_lot::Mutex;

pub type ResultHandle = Arc<Mutex<Option<Vec<u8>>>>;

#[derive(Clone, WasmerEnv)]
pub struct ArgsEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    args: Arc<Mutex<Arc<Vec<u8>>>>,
    result: Arc<Mutex<ResultHandle>>,
}

impl ArgsEnv {
    pub fn set_args(&self, args: Arc<Vec<u8>>) {
        let mut lock = self.args.lock();
        *lock = args;
    }

    pub fn set_result_handle(&self, new_hdl: ResultHandle) {
        let mut result = self.result.lock();
        *result = new_hdl;
    }
}

fn get_args(env: &ArgsEnv, len_out: WasmPtr<u64>) -> i64 {
    log::trace!("Got `get_args` call");

    let memory = env.memory.get_ref().unwrap();

    let args_lock = env.args.lock();
    let args = &*args_lock;

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

fn set_result(env: &ArgsEnv, buf_ptr: WasmPtr<u8, Array>, buf_len: u32) {
    log::debug!("Got result of size {}", buf_len);

    let result_outer_lock = env.result.lock();
    let result_outer = &*result_outer_lock;

    let mut result = result_outer.lock();

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

fn get_random_value(env: &ArgsEnv, buf_ptr: WasmPtr<u8, Array>, buf_len: u32) {
    log::trace!("Got \"get_random_value\" call with buffer size {buf_len}");

    let memory = env.memory.get_ref().unwrap();

    let buf_slice = unsafe {
        let buf_ptr = memory.view::<u8>().as_ptr().add(buf_ptr.offset() as usize) as *mut u8;
        std::slice::from_raw_parts_mut(buf_ptr, buf_len as usize)
    };

    let mut rng = rand::thread_rng();
    buf_slice
        .try_fill(&mut rng)
        .expect("Failed to fill buffer with random data");
}

pub fn get_imports(store: &Store, args: Arc<Vec<u8>>, result: ResultHandle) -> (Exports, ArgsEnv) {
    let args_env = ArgsEnv {
        args: Arc::new(Mutex::new(args)),
        result: Arc::new(Mutex::new(result)),
        memory: Default::default(),
        allocate: Default::default(),
    };

    let mut ns = Exports::new();
    ns.insert(
        "set_result",
        Function::new_native_with_env(store, args_env.clone(), set_result),
    );
    ns.insert(
        "get_args",
        Function::new_native_with_env(store, args_env.clone(), get_args),
    );
    ns.insert(
        "get_random_value",
        Function::new_native_with_env(store, args_env.clone(), get_random_value),
    );

    (ns, args_env)
}
