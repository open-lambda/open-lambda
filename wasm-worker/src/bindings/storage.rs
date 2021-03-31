use wasmer::{Array, LazyInit, Function, Memory, NativeFunc, Store, Exports, WasmPtr, WasmerEnv, Yielder};

use std::sync::{Arc, Mutex};

use lambda_store_client::Client as Database;

use open_lambda_protocol::{CollectionInfo, Operation};

#[ derive(Clone, WasmerEnv) ]
struct StorageEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name="internal_alloc_buffer"))] allocate: LazyInit<NativeFunc<u32, i64>>,
    database: Arc<Mutex<Option<Database>>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
}

fn get_collection_schema(env: &StorageEnv, name_ptr: WasmPtr<u8, Array>, name_len: u32, len_out: WasmPtr<u64>) -> i64 {
    let mut db_lock = env.database.lock().unwrap();

    let database = if let Some(inner) = db_lock.take() {
        inner
    } else {
        // Connect to lambda store
        lambda_store_client::create_client("localhost")
    };

    let memory = env.memory.get_ref().unwrap();
    let col_name = name_ptr.get_utf8_string(memory, name_len).unwrap();

    let col = database.get_collection(col_name).unwrap();

    let (key_type, fields) = col.get_schema().clone_inner();
    let info = CollectionInfo{ identifier: col.get_identifier(), key_type, fields };
    let data = bincode::serialize(&info).unwrap();

    let offset = env.allocate.get_ref().unwrap().call(data.len() as u32).unwrap();

    if offset < 0 {
        panic!("failed to allocate");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, data.len())
    };

    out_slice.clone_from_slice(&data.as_slice());

    let len = unsafe{ len_out.deref_mut(memory) }.unwrap();
    len.set(data.len() as u64);

    *db_lock = Some(database);

    offset
}

fn execute_operation(env: &StorageEnv, op_data: WasmPtr<u8, Array>, op_data_len: u32, len_out: WasmPtr<u64>) -> i64 {
    let memory = env.memory.get_ref().unwrap();
    let mut db_lock = env.database.lock().unwrap();
    let database = db_lock.take().expect("Database not initialized yet");

    let in_slice = unsafe {
        let offset = op_data.offset();
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts(raw_ptr, op_data_len as usize)
    };

    let yielder = env.yielder.get_ref().unwrap().get();

    let op: Operation = bincode::deserialize(in_slice).unwrap();
    let col_id = op.get_collection().unwrap();
    let col = database.get_collection_by_id(col_id).unwrap();

    let result = yielder.async_suspend(async move {
        col.execute_operation(op, None).await
    });

    let result_data = bincode::serialize(&result).unwrap();

    let offset = env.allocate.get_ref().unwrap().call(result_data.len() as u32).unwrap();

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, result_data.len())
    };

    out_slice.clone_from_slice(&result_data.as_slice());

    let len = unsafe{ len_out.deref_mut(memory) }.unwrap();
    len.set(result_data.len() as u64);

    *db_lock = Some(database);

    offset
}

pub fn get_imports(store: &Store) -> Exports {
    let mut ns = Exports::new();
    let env = StorageEnv {
        database: Arc::new(Mutex::new(None)),
        yielder: Default::default(),
        allocate: Default::default(),
        memory: Default::default()
    };

    ns.insert("get_collection_schema", Function::new_native_with_env(&store, env.clone(), get_collection_schema));
    ns.insert("execute_operation", Function::new_native_with_env(&store, env, execute_operation));

    ns
}
