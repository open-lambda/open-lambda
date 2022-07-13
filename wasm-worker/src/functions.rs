use dashmap::DashMap;

use std::fs;
use std::io::{Read, Write};
use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use parking_lot::Mutex;

use wasmer::{BaseTunables, ImportObject, Instance, Module, Pages, Store};
use wasmer_compiler_cranelift::Cranelift;
use wasmer_compiler_singlepass::Singlepass;
use wasmer_engine::Engine;
use wasmer_engine_dylib::Dylib as NativeEngine;

use crate::WasmCompilerType;

use crate::bindings::{
    self,
    args::{ArgsEnv, ResultHandle},
    ipc::IpcEnv,
};

#[cfg(feature = "llvm-backend")]
use wasmer_compiler_llvm::LLVM;

const MAX_IDLE_INSTANCES: usize = 20;

type IdleInstancesList = Mutex<Vec<InstanceData>>;

pub struct Function {
    module: Module,
    next_instance_id: Arc<AtomicU64>,
    store: Arc<Store>,
    idle_list: Arc<IdleInstancesList>,
}

struct InstanceData {
    identifier: u64,
    instance: Instance,
    args_env: ArgsEnv,
    #[allow(dead_code)]
    ipc_env: IpcEnv,
}

pub struct InstanceHandle {
    idle_list: Arc<IdleInstancesList>,
    data: InstanceData,
}

impl Function {
    pub fn get_idle_instance(
        &self,
        args: Arc<Vec<u8>>,
        addr: SocketAddr,
        result_hdl: ResultHandle,
    ) -> InstanceHandle {
        {
            if let Some(data) = self.idle_list.lock().pop() {
                log::trace!("Reusing WASM instance with id={}", data.identifier);

                data.args_env.set_args(args);
                data.args_env.set_result_handle(result_hdl);

                return InstanceHandle {
                    idle_list: self.idle_list.clone(),
                    data,
                };
            }
        }

        let identifier = self.next_instance_id.fetch_add(1, Ordering::SeqCst);

        log::trace!("Creating new WASM instance with id={identifier}");

        let mut import_object = ImportObject::new();

        let (args_imports, args_env) = bindings::args::get_imports(&self.store, args, result_hdl);
        let log_imports = bindings::log::get_imports(&self.store);
        let (ipc_imports, ipc_env) = bindings::ipc::get_imports(&self.store, addr);

        import_object.register("ol_args", args_imports);
        import_object.register("ol_log", log_imports);
        import_object.register("ol_ipc", ipc_imports);

        let instance =
            Instance::new(&self.module, &import_object).expect("failed to create instance");

        let data = InstanceData {
            identifier,
            instance,
            args_env,
            ipc_env,
        };

        InstanceHandle {
            data,
            idle_list: self.idle_list.clone(),
        }
    }
}

impl InstanceHandle {
    pub fn get(&self) -> &Instance {
        &self.data.instance
    }

    pub fn mark_idle(self) {
        let mut idle_list = self.idle_list.lock();

        if idle_list.len() < MAX_IDLE_INSTANCES {
            log::trace!(
                "Putting instance with id={} back into idle list",
                self.data.identifier
            );
            idle_list.push(self.data);
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
    store: Arc<Store>,
    functions: Arc<DashMap<String, Arc<Function>>>,
    compiler_type: WasmCompilerType,
}

impl FunctionManager {
    pub async fn new(compiler_type: WasmCompilerType) -> Self {
        let engine = match compiler_type {
            WasmCompilerType::Cranelift => {
                log::info!("Using Cranelift compiler. Might result in lower performance");
                NativeEngine::new(Cranelift::default()).engine()
            }
            WasmCompilerType::LLVM => {
                cfg_if::cfg_if! {
                    if #[cfg(feature="llvm-backend") ] {
                        log::info!("Using LLVM compiler");
                        NativeEngine::new(LLVM::default()).engine()
                    } else {
                        panic!("LLVM backend is disabled");
                    }
                }
            }
            WasmCompilerType::Singlepass => {
                log::info!("Using Singlepass compiler. Might result in lower performance.");
                NativeEngine::new(Singlepass::default()).engine()
            }
        };

        // Always use dynamic memory so we can clone the zygote
        let mut tunables = BaseTunables::for_target(engine.target());
        tunables.static_memory_bound = Pages(0);

        let store = Arc::new(Store::new_with_tunables(&engine, tunables));
        let functions = Arc::new(DashMap::new());

        Self {
            functions,
            store,
            compiler_type,
        }
    }

    pub fn get_compiler_type(&self) -> &WasmCompilerType {
        &self.compiler_type
    }

    pub async fn get_function(&self, name: &str) -> Option<Arc<Function>> {
        self.functions.get(name).map(|entry| entry.value().clone())
    }

    pub async fn load_function(&self, path: PathBuf, cache_path: PathBuf) {
        let store = self.store.clone();
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

            log::info!("Loaded cached version of program \"{name}\"");

            unsafe { Module::deserialize(&store, &binary).expect("Failed to deserialize module") }
        } else {
            let mut code = Vec::new();
            file.read_to_end(&mut code).unwrap();

            // Load and compile
            let module = match Module::new(&self.store, code) {
                Ok(module) => {
                    log::info!("Compiled program \"{name}\"");
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
            next_instance_id: Arc::new(AtomicU64::new(1)),
            idle_list: Default::default(),
            store,
            module,
        });
        self.functions.insert(name, function);
    }
}
