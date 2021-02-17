use wasmer::{Store, Module, Engine};
use wasmer_compiler_llvm::LLVM;
use wasmer_engine_native::Native as NativeEngine;

use std::sync::Arc;

pub struct Program {
    pub module: Arc<Module>,
    store: Arc<Store>,
}

pub struct ProgramManager {
    store: Arc<Store>,

    #[ allow(dead_code) ]
    engine: Box<dyn Engine>
}

impl ProgramManager {
    pub fn new() -> Self {
        let compiler = LLVM::default();
        let engine = Box::new( NativeEngine::new(compiler).engine() );
        let store = Arc::new(Store::new(&*engine));

        Self{ store, engine }
    }
}
