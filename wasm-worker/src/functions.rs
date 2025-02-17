use std::collections::HashMap;
use std::fs;
use std::io::{Read, Write};
use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};

use anyhow::Context;

use dashmap::DashMap;

use wasmtime::{AsContextMut, Engine, Instance, Linker, Module, Store};

use crate::bindings::{self, BindingsData, args::ResultHandle};

const MAX_IDLE_INSTANCES: usize = 100;

pub type InstanceId = u64;

type IdleInstancesList = crossbeam::queue::SegQueue<InstanceData>;

pub struct Function {
    next_instance_id: Arc<AtomicU64>,
    idle_list: Arc<IdleInstancesList>,
    engine: Arc<Engine>,
    module: Arc<Module>,
}

struct InstanceData {
    identifier: InstanceId,
    store: Store<BindingsData>,
    instance: Instance,
}

pub struct InstanceHandle {
    idle_list: Arc<IdleInstancesList>,
    data: InstanceData,
}

impl Function {
    pub async fn get_idle_instance(
        &self,
        args: Vec<u8>,
        config_values: &HashMap<String, String>,
        addr: SocketAddr,
        result_hdl: ResultHandle,
    ) -> InstanceHandle {
        if let Some(mut data) = self.idle_list.pop() {
            log::trace!("Reusing WASM instance with id={}", data.get_identifier());
            data.refresh(config_values, addr, args, result_hdl);

            InstanceHandle::new(self.idle_list.clone(), data)
        } else {
            let identifier = self.next_instance_id.fetch_add(1, Ordering::SeqCst);

            log::trace!("Creating new WASM instance with id={identifier}");

            let data = InstanceData::new(
                &self.engine,
                &self.module,
                identifier,
                config_values.clone(),
                addr,
                args,
                result_hdl,
            )
            .await;

            InstanceHandle::new(self.idle_list.clone(), data)
        }
    }
}

impl InstanceHandle {
    fn new(idle_list: Arc<IdleInstancesList>, data: InstanceData) -> Self {
        Self { idle_list, data }
    }

    pub fn get(&mut self) -> (&mut Store<BindingsData>, &Instance) {
        (&mut self.data.store, &self.data.instance)
    }

    pub fn mark_idle(self) {
        if self.idle_list.len() < MAX_IDLE_INSTANCES {
            log::trace!(
                "Putting instance with id={} back into idle list",
                self.data.identifier
            );
            self.idle_list.push(self.data);
        } else {
            log::trace!(
                "Discarding instance with id={}; idle list is already full",
                self.data.identifier
            );
        }
    }

    #[allow(dead_code)]
    pub fn discard(self) {
        log::trace!(
            "Discarding instance with id={} as requested",
            self.data.identifier
        );
    }
}

pub struct FunctionManager {
    functions: Arc<DashMap<String, Arc<Function>>>,
    next_instance_id: Arc<AtomicU64>,
    engine: Arc<wasmtime::Engine>,
}

impl FunctionManager {
    pub async fn new() -> anyhow::Result<Self> {
        let next_instance_id = Arc::new(AtomicU64::new(1));
        let mut config = wasmtime::Config::new();
        config.async_support(true);

        // Optimize for performance
        config.cranelift_opt_level(wasmtime::OptLevel::Speed);
        config.allocation_strategy(wasmtime::InstanceAllocationStrategy::pooling());

        let engine =
            wasmtime::Engine::new(&config).with_context(|| "Failed to create wasmtime engine")?;

        Ok(Self {
            functions: Default::default(),
            engine: Arc::new(engine),
            next_instance_id,
        })
    }

    pub async fn get_function(&self, name: &str) -> Option<Arc<Function>> {
        self.functions.get(name).map(|entry| entry.value().clone())
    }

    pub async fn load_function(&self, path: PathBuf, cache_path: PathBuf) {
        let os_name = path.file_stem().unwrap();
        let name = String::from(os_name.to_str().unwrap());

        let cpath = cache_path.join(format!("{name}.bin"));

        log::debug!("Loading function \"{name}\" with path {path:?}");
        let mut file = match fs::File::open(&path) {
            Ok(f) => f,
            Err(err) => {
                log::error!("Failed to open function file at \"{path:?}\": {err}");
                std::process::exit(1);
            }
        };

        let is_cached = {
            let file_meta = fs::metadata(&path).expect("Failed to read file metadata");

            match fs::metadata(&cpath) {
                Ok(cache_meta) => cache_meta.modified().unwrap() > file_meta.modified().unwrap(),
                Err(err) => {
                    if err.kind() == std::io::ErrorKind::NotFound {
                        false
                    } else {
                        panic!("Failed to read cachefile metadata: {err}");
                    }
                }
            }
        };

        let module = if is_cached {
            // Load cached data
            let mut file = fs::File::open(cpath).unwrap();

            let mut binary = Vec::new();
            file.read_to_end(&mut binary).unwrap();

            log::info!("Loaded cached version of function \"{name}\"");

            unsafe {
                Module::deserialize(&self.engine, &binary).expect("Failed to deserialize module")
            }
        } else {
            let mut code = Vec::new();
            file.read_to_end(&mut code).unwrap();

            // Load and compile
            let module = match Module::new(&self.engine, code) {
                Ok(module) => {
                    log::info!("Compiled fucntion \"{name}\"");
                    module
                }
                Err(err) => panic!("Failed to compile wasm file \"{name}\": {err:?}"),
            };

            let binary = module.serialize().unwrap();

            // Cache binary
            if let Err(err) = fs::create_dir(&cache_path) {
                if err.kind() != std::io::ErrorKind::AlreadyExists {
                    panic!(
                        "Failed to create program cache directory at '{}': {err}",
                        cache_path.display()
                    );
                }
            }

            {
                //FIXME add a header with wasmer/compiler version and checksum

                let mut cache_file = fs::File::create(&cpath).expect("Failed to create cache file");
                cache_file
                    .write_all(&binary)
                    .expect("Failed to write cache file");
                log::debug!("Stored cached program at \"{}\"", cpath.display());
            }

            module
        };

        let function = Arc::new(Function {
            engine: self.engine.clone(),
            module: Arc::new(module),
            next_instance_id: self.next_instance_id.clone(),
            idle_list: Default::default(),
        });
        self.functions.insert(name, function);
    }
}

impl InstanceData {
    #[allow(clippy::too_many_arguments)]
    pub(super) async fn new(
        engine: &Engine,
        module: &Module,
        identifier: InstanceId,
        config_values: HashMap<String, String>,
        addr: SocketAddr,
        args: Vec<u8>,
        result_hdl: ResultHandle,
    ) -> Self {
        let mut linker = Linker::new(engine);

        let data = bindings::BindingsData::new(addr, config_values, args, result_hdl);

        bindings::args::get_imports(&mut linker);
        bindings::log::get_imports(&mut linker);
        bindings::ipc::get_imports(&mut linker);
        bindings::config::get_imports(&mut linker);

        let mut store = Store::new(engine, data);

        let instance = linker
            .instantiate_async(store.as_context_mut(), module)
            .await
            .expect("Failed to create instance");

        let _ = instance
            .get_func(&mut store, "_initialize_instance")
            .unwrap()
            .call_async(&mut store, &[], &mut [])
            .await;

        Self {
            identifier,
            instance,
            store,
        }
    }

    /// Make the InstanceData ready to be used for another job
    pub(super) fn refresh(
        &mut self,
        _config_values: &HashMap<String, String>,
        _addr: SocketAddr,
        args: Vec<u8>,
        result_hdl: ResultHandle,
    ) {
        let bindings = self.store.data_mut();

        bindings.args.set_args(args);
        bindings.args.set_result_handle(result_hdl);
    }

    pub fn get_identifier(&self) -> InstanceId {
        self.identifier
    }
}
