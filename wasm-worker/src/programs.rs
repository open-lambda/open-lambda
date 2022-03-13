use wasmer::{Module, Store};
#[cfg(feature = "cranelift")]
use wasmer_compiler_cranelift::Cranelift;
#[cfg(not(feature = "cranelift"))]
use wasmer_compiler_llvm::LLVM;
use wasmer_engine_dylib::Dylib as NativeEngine;

use std::sync::Arc;
use tokio::sync::Mutex;

use dashmap::DashMap;

use crate::condvar::Condvar;

enum ProgramState {
    Empty,
    Loading,
    Loaded(Arc<Program>),
}

pub struct Program {
    pub module: Arc<Module>,
    pub store: Arc<Store>,
}

pub struct ProgramHandle {
    state: Mutex<ProgramState>,
    condition: Condvar,
}

impl ProgramHandle {
    fn empty() -> Self {
        Self {
            state: Mutex::new(ProgramState::Empty),
            condition: Condvar::new(),
        }
    }
}

pub struct ProgramManager {
    store: Arc<Store>,
    programs: DashMap<String, ProgramHandle>,
}

impl ProgramManager {
    pub fn new() -> Self {
        #[cfg(feature = "cranelift")]
        let compiler = Cranelift::default();
        #[cfg(not(feature = "cranelift"))]
        let compiler = LLVM::default();

        let engine = Box::new(NativeEngine::new(compiler).engine());
        let store = Arc::new(Store::new(&*engine));
        let programs = DashMap::new();

        Self { store, programs }
    }

    pub async fn get_program(&self, name: String) -> Arc<Program> {
        let path = format!("./test-registry.wasm/{}.wasm", name);
        let e = self.programs.entry(name).or_insert(ProgramHandle::empty());

        let mut state = e.value().state.lock().await;

        loop {
            match &*state {
                ProgramState::Empty => {
                    *state = ProgramState::Loading;
                    break;
                }
                ProgramState::Loading => {
                    state = e.value().condition.wait(state, &e.value().state).await;
                }
                ProgramState::Loaded(p) => {
                    return p.clone();
                }
            }
        }

        drop(state);

        // Load and compile
        match Module::from_file(&*self.store, &path) {
            Ok(m) => {
                let p = Arc::new(Program {
                    module: Arc::new(m),
                    store: self.store.clone(),
                });

                let mut state = e.value().state.lock().await;
                *state = ProgramState::Loaded(p.clone());
                e.value().condition.notify_all(state);

                log::info!("Loaded new program `{}`", path);

                p
            }
            Err(e) => panic!("Failed to compile wasm `{}`: {:?}", path, e),
        }
    }
}
