use wasmer::{
    Array, Exports, Function, LazyInit, Memory, NativeFunc, Store, WasmPtr, WasmerEnv, Yielder,
};

use std::net::SocketAddr;
use std::sync::Arc;

use parking_lot::Mutex;

use serde_bytes::ByteBuf;

use lambda_store_client::Client as Database;

use hyper::client::connect::HttpConnector;
use hyper::client::Client as HttpClient;
use hyper::{Body, Request};

use open_lambda_protocol::{BatchCallData, BatchCallResult};

use crate::bindings::storage::TransactionHandle;

#[derive(Clone, WasmerEnv)]
pub struct IpcEnv {
    #[wasmer(export)]
    memory: LazyInit<Memory>,
    #[wasmer(export(name = "internal_alloc_buffer"))]
    allocate: LazyInit<NativeFunc<u32, i64>>,
    #[wasmer(yielder)]
    yielder: LazyInit<Yielder>,
    database: Arc<Database>,
    transaction: Arc<Mutex<TransactionHandle>>,
    addr: SocketAddr,
}

fn batch_call(
    env: &IpcEnv,
    call_data_ptr: WasmPtr<u8, Array>,
    call_data_len: u32,
    len_out: WasmPtr<u64>,
) -> i64 {
    log::trace!("Got `batch_call` call");

    let memory = env.memory.get_ref().unwrap();
    let yielder = env.yielder.get_ref().unwrap().get();

    // First commit the transaction to prevent deadlocks
    // TODO we should have a way to enable "linearizable"-mode
    {
        let tx_outer_lock = env.transaction.lock();
        let mut tx_lock = tx_outer_lock.lock();

        if let Some(tx) = tx_lock.take() {
            let result = yielder.async_suspend(tx.commit());

            if let Err(err) = result {
                panic!("Transaction failed: {err}");
            }
        }

        // Create a new transaction in case there is more
        *tx_lock = Some(env.database.begin_transaction());
    }

    // This sets up a connection pool that will be reused
    lazy_static::lazy_static! {
        static ref HTTP_CLIENT: HttpClient<HttpConnector, Body> = HttpClient::new();
    };

    let call_data: BatchCallData = unsafe {
        let ptr = memory
            .view::<u8>()
            .as_ptr()
            .add(call_data_ptr.offset() as usize) as *mut u8;
        let len = call_data_len as usize;

        let raw_data = std::slice::from_raw_parts(ptr, len);
        bincode::deserialize(raw_data).expect("Failed to parse call data")
    };

    let mut results: BatchCallResult = vec![];

    for (object_id, function_id, args) in call_data.into_iter() {
        let result = yielder.async_suspend(async move {
            let object_type = match env.database.get_object(object_id).await {
                Ok(object) => object.get_object_type(),
                Err(err) => {
                    panic!("Failed to get object: {err}");
                }
            };

            let metadata = match object_type.get_function_by_id(&function_id) {
                Some(meta) => meta,
                None => {
                    panic!("No such function");
                }
            };

            let oid_hex = object_id
                .to_hex_string()
                .replace('#', "%23")
                .replace(':', "%3A");

            let uri = hyper::Uri::builder()
                .scheme("http")
                .authority(format!("{}", env.addr))
                .path_and_query(format!("/run_on/{}?object_id={oid_hex}", metadata.name))
                .build()
                .unwrap();

            let request = Request::builder()
                .header("User-Agent", "open-lambda-wasm/1.0")
                .method("POST")
                .uri(uri)
                .body(args.into())
                .unwrap();

            let response = match HTTP_CLIENT.request(request).await {
                Ok(resp) => resp,
                Err(err) => {
                    panic!("Internal call to {} failed: {err}", env.addr);
                }
            };

            if !response.status().is_success() {
                panic!(
                    "Request was unsuccessful. Go status code: {}",
                    response.status()
                );
            }

            let buf = hyper::body::to_bytes(response).await.unwrap();
            Ok(ByteBuf::from(buf.to_vec()))
        });

        results.push(result);
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
    addr: SocketAddr,
    database: Arc<Database>,
    transaction: TransactionHandle,
) -> (Exports, IpcEnv) {
    let mut ns = Exports::new();
    let env = IpcEnv {
        memory: Default::default(),
        allocate: Default::default(),
        yielder: Default::default(),
        database,
        transaction: Arc::new(Mutex::new(transaction)),
        addr,
    };

    ns.insert(
        "batch_call",
        Function::new_native_with_env(store, env.clone(), batch_call),
    );

    (ns, env)
}
