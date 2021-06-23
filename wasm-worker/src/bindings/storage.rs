use wasmer::{Array, LazyInit, Function, Memory, NativeFunc, Store, Exports, WasmPtr, WasmerEnv, Yielder};

use std::sync::{Arc, Mutex};

use lambda_store_client::Client as Database;
use lambda_store_client::Transaction;

use open_lambda_protocol::{CollectionInfo, Operation};

#[ derive(Default, Clone, WasmerEnv) ]
pub struct StorageEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name="internal_alloc_buffer"))] allocate: LazyInit<NativeFunc<u32, i64>>,
    database: Arc<Mutex<Option<Database>>>,
    transaction: Arc<Mutex<Option<Transaction>>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
}

impl StorageEnv {
    pub async fn commit(&self) -> bool {
        let tx = {
            let mut tx_lock = self.transaction.lock().unwrap();
            if let Some(tx) = tx_lock.take() {
                tx
            } else {
                // Nothing to commit
                return true;
            }
        };

        tx.commit().await.is_ok()
    }
}

fn get_collection_schema(env: &StorageEnv, name_ptr: WasmPtr<u8, Array>, name_len: u32, len_out: WasmPtr<u64>) -> i64 {
    let mut db_lock = env.database.lock().unwrap();
    let lock_inner = db_lock.take();

    let yielder = env.yielder.get_ref().unwrap().get();

    let database = yielder.async_suspend(async move {
        if let Some(inner) = lock_inner {
            inner
        } else {
            // Connect to lambda store
            lambda_store_client::create_client("localhost").await
        }
    });

    let memory = env.memory.get_ref().unwrap();
    let col_name = name_ptr.get_utf8_string(memory, name_len).unwrap();

    let col = database.get_collection(col_name).expect("No such collection");

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

    out_slice.clone_from_slice(data.as_slice());

    let len = unsafe{ len_out.deref_mut(memory) }.unwrap();
    len.set(data.len() as u64);

    *db_lock = Some(database);

    offset
}

fn execute_operation(env: &StorageEnv, op_data: WasmPtr<u8, Array>, op_data_len: u32, len_out: WasmPtr<u64>) -> i64 {
    let memory = env.memory.get_ref().unwrap();
    let mut db_lock = env.database.lock().unwrap();
    let database = db_lock.take().expect("Database not initialized yet");

    let mut tx_lock = env.transaction.lock().unwrap();

    let tx = if let Some(tx) = tx_lock.take() {
        tx
    } else {
        database.begin_transaction()
    };

    let op_slice = unsafe {
        let offset = op_data.offset();
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts(raw_ptr, op_data_len as usize)
    };

    let yielder = env.yielder.get_ref().unwrap().get();

    let op: Operation = bincode::deserialize(op_slice).unwrap();
    let col_id = op.get_collection().unwrap();
    let col = tx.get_collection_by_id(col_id).expect("No such collection");

    log::debug!("Executing operation: {:?}", op);

    let result = yielder.async_suspend(async move {
        let ntype = if op.is_write() {
            lambda_store_client::NodeType::Head
        } else {
            lambda_store_client::NodeType::Tail
        };

        col.execute_operation(op, ntype).await
    });

    log::debug!("Op result is: {:?}", result);

    let result_data = bincode::serialize(&result).expect("Failed to serialize OpResult");

    let offset = env.allocate.get_ref().unwrap().call(result_data.len() as u32).unwrap();

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, result_data.len())
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = unsafe{ len_out.deref_mut(memory) }.unwrap();
    len.set(result_data.len() as u64);

    *db_lock = Some(database);
    *tx_lock = Some(tx);

    offset
}

pub fn get_imports(store: &Store, env: StorageEnv) -> Exports {
    let mut ns = Exports::new();
    ns.insert("get_collection_schema", Function::new_native_with_env(store, env.clone(), get_collection_schema));
    ns.insert("execute_operation", Function::new_native_with_env(store, env, execute_operation));

    ns
}
