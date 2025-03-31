use std::future::Future;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

use parking_lot::Mutex;

use rand::TryRngCore;

use wasmtime::{Caller, Linker, Val};

use super::{BindingsData, fill_slice, get_slice, get_slice_mut, set_u64};

pub type ResultHandle = Arc<Mutex<Option<Vec<u8>>>>;

pub struct ArgsData {
    args: Vec<u8>,
    result: ResultHandle,
}

impl ArgsData {
    pub fn new(args: Vec<u8>, result: ResultHandle) -> Self {
        Self { args, result }
    }

    pub fn set_args(&mut self, args: Vec<u8>) {
        self.args = args;
    }

    pub fn set_result_handle(&mut self, new_hdl: ResultHandle) {
        self.result = new_hdl;
    }
}

fn get_args(
    mut caller: Caller<'_, BindingsData>,
    args: (i32,),
) -> Box<dyn Future<Output = i64> + Send + '_> {
    let (len_out,) = args;

    Box::new(async move {
        log::trace!("Got \"get_args\" call");

        let alloc_fn = caller
            .get_export("internal_alloc_buffer")
            .expect("Missing alloc export")
            .into_func()
            .expect("Not a function");

        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        //FIXME do not copy
        let args = caller.data().args.args.clone();

        if args.is_empty() {
            return 0;
        }

        let mut result = vec![Val::I64(0)];

        alloc_fn
            .call_async(&mut caller, &[Val::I32(args.len() as i32)], &mut result)
            .await
            .unwrap();

        let offset = result[0].i64().unwrap() as i32;
        fill_slice(&caller, &memory, offset, args.as_slice());
        set_u64(&caller, &memory, len_out, args.len() as u64);

        offset as i64
    })
}

fn get_unix_time() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("System time before UNIX epoch")
        .as_secs()
}

fn set_result(mut caller: Caller<'_, BindingsData>, buf_ptr: i32, buf_len: u32) {
    log::trace!("Got \"set_result\" call with result of size {buf_len}");

    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();

    let data = &caller.data().args;
    let mut result = data.result.lock();

    if result.is_some() {
        panic!("Result was already set");
    }

    let buf_slice = get_slice(&caller, &memory, buf_ptr, buf_len);

    let mut vec = Vec::new();
    vec.extend_from_slice(buf_slice);

    *result = Some(vec);
}

fn get_random_value(mut caller: Caller<'_, BindingsData>, buf_ptr: i32, buf_len: u32) {
    log::trace!("Got \"get_random_value\" call with buffer size {buf_len}");

    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();

    let buf_slice = get_slice_mut(&caller, &memory, buf_ptr, buf_len);

    let mut rng = rand::rng();
    rng.try_fill_bytes(buf_slice)
        .expect("Failed to fill buffer with random data");
}

pub fn get_imports(linker: &mut Linker<BindingsData>) {
    let module = "ol_args";

    linker
        .func_wrap(module, "get_unix_time", get_unix_time)
        .unwrap();
    linker.func_wrap(module, "set_result", set_result).unwrap();
    linker
        .func_wrap_async(module, "get_args", get_args)
        .unwrap();
    linker
        .func_wrap(module, "get_random_value", get_random_value)
        .unwrap();
}
