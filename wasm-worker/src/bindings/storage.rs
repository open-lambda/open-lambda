use wasmer::{
    Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv, Yielder,
};

use std::sync::{Arc, Mutex};

use lambda_store_client::Client as Database;
use lambda_store_client::Transaction;

use open_lambda_protocol::DataOperation;

#[derive(Clone, WasmerEnv)]
pub struct StorageEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    database: Arc<Database>,
    transaction: Arc<Mutex<Option<Transaction>>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
}

impl StorageEnv {
    pub fn new(database: Arc<Database>) -> Self {
        Self {
            database,
            memory: Default::default(),
            allocate: Default::default(),
            transaction: Default::default(),
            yielder: Default::default(),
        }
    }

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

fn get_configuration(env: &StorageEnv, len_out: WasmPtr<u64>) -> i64 {
    log::trace!("Got `get_configuration` call");

    let memory = env.memory.get_ref().unwrap();

    let object_types = env.database.get_object_types();
    let data = bincode::serialize(&object_types).unwrap();

    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(data.len() as u32)
        .unwrap();

    if offset < 0 {
        panic!("failed to allocate");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, data.len())
    };

    out_slice.clone_from_slice(data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(data.len() as u64);

    offset
}

fn execute_operation(
    env: &StorageEnv,
    op_data: WasmPtr<u8, Array>,
    op_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    let memory = env.memory.get_ref().unwrap();
    let mut tx_lock = env.transaction.lock().unwrap();

    if tx_lock.is_none() {
        *tx_lock = Some(env.database.begin_transaction())
    }

    let tx = tx_lock.as_mut().unwrap();

    let op_slice = unsafe {
        let offset = op_data.offset();
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts(raw_ptr, op_data_len as usize)
    };

    let yielder = env.yielder.get_ref().unwrap().get();

    let op: DataOperation = bincode::deserialize(op_slice).unwrap();

    log::debug!("Executing operation: {op:?}");
    let result = yielder.async_suspend(async move { tx.execute_operation(op).await });

    log::debug!("Op result is: {result:?}");

    let result_data = bincode::serialize(&result).expect("Failed to serialize OpResult");

    let offset = env
        .allocate
        .get_ref()
        .unwrap()
        .call(result_data.len() as u32)
        .unwrap();

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr().add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, result_data.len())
    };

    out_slice.clone_from_slice(result_data.as_slice());

    let len = len_out.deref(memory).unwrap();
    len.set(result_data.len() as u64);

    offset
}

pub fn get_imports(store: &Store, env: StorageEnv) -> Exports {
    let mut ns = Exports::new();
    ns.insert(
        "get_configuration",
        Function::new_native_with_env(store, env.clone(), get_configuration),
    );
    ns.insert(
        "execute_operation",
        Function::new_native_with_env(store, env, execute_operation),
    );

    ns
}
