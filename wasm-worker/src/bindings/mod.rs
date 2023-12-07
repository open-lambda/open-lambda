use wasmtime::{Caller, Memory, Val};

use std::collections::HashMap;
use std::net::SocketAddr;

pub mod args;
pub mod config;
pub mod ipc;
pub mod log;

use args::ResultHandle;

/// All bindings data for a specific instance
pub struct BindingsData {
    pub args: args::ArgsData,
    pub ipc: ipc::IpcData,
    pub config: config::ConfigData,
}

impl BindingsData {
    pub fn new(
        addr: SocketAddr,
        config_values: HashMap<String, String>,
        args: Vec<u8>,
        result: ResultHandle,
    ) -> Self {
        Self {
            ipc: ipc::IpcData::new(addr),
            args: args::ArgsData::new(args, result),
            config: config::ConfigData::new(config_values),
        }
    }
}

fn get_slice<'a>(
    caller: &Caller<'a, BindingsData>,
    memory: &Memory,
    offset: i32,
    len: u32,
) -> &'a [u8] {
    unsafe {
        let buf_ptr = memory.data_ptr(caller).offset(offset as isize) as *const u8;
        std::slice::from_raw_parts(buf_ptr, len as usize)
    }
}

fn get_slice_mut<'a>(
    caller: &Caller<'a, BindingsData>,
    memory: &Memory,
    offset: i32,
    len: u32,
) -> &'a mut [u8] {
    unsafe {
        let buf_ptr = memory.data_ptr(caller).offset(offset as isize);
        std::slice::from_raw_parts_mut(buf_ptr, len as usize)
    }
}

fn fill_slice(caller: &Caller<'_, BindingsData>, memory: &Memory, offset: i32, data: &[u8]) {
    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr(caller).add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, data.len())
    };

    out_slice.clone_from_slice(data);

    if offset < 0 {
        panic!("failed to allocate");
    }

    let out_slice = unsafe {
        let raw_ptr = memory.data_ptr(caller).add(offset as usize);
        std::slice::from_raw_parts_mut(raw_ptr, data.len())
    };

    out_slice.clone_from_slice(data);
}

fn get_str<'a>(
    caller: &Caller<'a, BindingsData>,
    memory: &Memory,
    offset: i32,
    len: u32,
) -> &'a str {
    unsafe {
        let raw_ptr = memory.data_ptr(caller).add(offset as usize);
        let slice = std::slice::from_raw_parts(raw_ptr, len as usize);
        std::str::from_utf8(slice).unwrap()
    }
}

fn set_u64(caller: &Caller<'_, BindingsData>, memory: &Memory, offset: i32, value: u64) {
    unsafe {
        let ptr = memory.data_ptr(caller).offset(offset as isize) as *mut u64;
        *ptr = value;
    }
}

async fn call_allocate(caller: &mut Caller<'_, BindingsData>, size: u32) -> i32 {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();

    let alloc_fn = caller
        .get_export("internal_alloc_buffer")
        .expect("Missing alloc export")
        .into_func()
        .expect("Not a function");

    let mut result = vec![Val::I64(0)];

    alloc_fn
        .call_async(&mut *caller, &[Val::I32(size as i32)], &mut result)
        .await
        .unwrap();

    let offset = result[0].i64().unwrap() as i32;

    if offset < 0 {
        panic!("Failed to allocate");
    }

    if (offset as usize) + (size as usize) > memory.data_size(caller) {
        panic!("Invalid pointer");
    }

    offset
}
