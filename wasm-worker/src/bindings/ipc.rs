use wasmer::{
    Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv, Yielder,
};

use std::sync::Arc;

use lambda_store_client::Client as Database;

use open_lambda_protocol::{BatchCallData, BatchCallResult};

#[derive(Clone, WasmerEnv)]
pub struct IpcEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
    database: Arc<Database>,
}

fn batch_call(
    env: &IpcEnv,
    call_data_ptr: WasmPtr<u8, Array>,
    call_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `batch_call` call");

    // Right now, for pachinko, this assumes that the calls are child calls
    // If we ever introduce parent, or sibling, calls things might need to be changed

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    let mut call_data: BatchCallData = unsafe {
        let ptr = memory
            .view::<u8>()
            .as_ptr()
            .add(call_data_ptr.offset() as usize) as *mut u8;
        let len = call_data_len as usize;

        let raw_data = std::slice::from_raw_parts(ptr, len);
        bincode::deserialize(raw_data).expect("Failed to parse call data")
    };

    let mut results: BatchCallResult = vec![];

    for (object_id, function_id, args) in call_data.drain(..) {
        let object_type = yielder.async_suspend(async move {
            match env.database.get_object(object_id).await {
                Ok(object) => object.get_object_type(),
                Err(err) => {
                    panic!("Failed to get object: {err}");
                }
            }
        });

        let metadata = match object_type.get_function_by_id(&function_id) {
            Some(meta) => meta,
            None => {
                log::error!("No such function");
                return -1;
            }
        };

        todo!();
    }

    let result_data = bincode::serialize(&results).unwrap();
    let buffer_len = result_data.len();
    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(buffer_len as u32)
        .unwrap();

    if offset < 0 {
        panic!("Failed to allocate");
    }

    if (offset as u64) + (buffer_len as u64) > memory.data_size() {
        panic!("Invalid pointer");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, buffer_len)
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(buffer_len as u64);

    offset
}

pub fn get_imports(
    store: &Store,
    database: Arc<Database>,
) -> Exports {
    let mut ns = Exports::new();
    let env = IpcEnv {
        memory: Default::default(),
        allocate: Default::default(),
        yielder: Default::default(),
        database,
    };

    ns.insert(
        "batch_call",
        Function::new_native_with_env(store, env.clone(), batch_call),
    );

    ns
}
