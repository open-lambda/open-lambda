use dashmap::DashMap;

use std::fs;
use std::io::{Read, Write};
use std::path::PathBuf;
use std::sync::Arc;

use lambda_store_utils::WasmCompilerType;
use open_lambda_protocol::ObjectTypeId;

use wasmer::{BaseTunables, Module, Pages, Store};
use wasmer_compiler_cranelift::Cranelift;
use wasmer_compiler_singlepass::Singlepass;
use wasmer_engine::Engine;
use wasmer_engine_dylib::Dylib as NativeEngine;

#[cfg(feature = "llvm-backend")]
use wasmer_compiler_llvm::LLVM;

pub struct ObjectFunctions {
    pub store: Arc<Store>,
    pub module: Module,
}

pub struct FunctionManager {
    functions: Arc<DashMap<ObjectTypeId, Arc<ObjectFunctions>>>,
    store: Arc<Store>,
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

    pub async fn get_object_functions(
        &self,
        object_type: &ObjectTypeId,
    ) -> Option<Arc<ObjectFunctions>> {
        self.functions
            .get(object_type)
            .map(|entry| entry.value().clone())
    }

    pub async fn load_object_functions(
        &self,
        object_type: ObjectTypeId,
        path: PathBuf,
        cache_path: PathBuf,
    ) {
        let store = self.store.clone();
        let os_name = path.file_stem().unwrap();
        let name = String::from(os_name.to_str().unwrap());

        let cpath = cache_path.join(format!("{name}.bin"));

        log::debug!("Loading function \"{name}\" with path {path:?}");
        let mut file = match fs::File::open(&path) {
            Ok(f) => f,
            Err(err) => {
                log::error!("Failed to open function file at `{path:?}`: {err}");
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

            unsafe { Module::deserialize(&*store, &binary).expect("Failed to deserialize module") }
        } else {
            let mut code = Vec::new();
            file.read_to_end(&mut code).unwrap();

            // Load and compile
            let module = match Module::new(&*self.store, code) {
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

        let functions = Arc::new(ObjectFunctions { store, module });
        self.functions.insert(object_type, functions);
    }
}
