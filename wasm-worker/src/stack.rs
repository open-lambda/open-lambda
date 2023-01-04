use async_wormhole::stack::Stack;
use wasmer_vm::Mmap;

use crossbeam::queue::SegQueue;

pub struct MmapStack {
    mmap: Mmap,
}

impl Stack for MmapStack {
    fn new() -> Result<Self, std::io::Error> {
        let mmap = Mmap::with_at_least(8 * 1024 * 1024).unwrap();
        Ok(Self { mmap })
    }

    fn top(&self) -> *mut usize {
        self.mmap.as_ptr() as *mut usize
    }

    // Stack grows down on unix systems
    fn bottom(&self) -> *mut usize {
        let len = self.mmap.len() as isize;
        let ptr = self.mmap.as_ptr();

        unsafe { ptr.offset(len) as *mut usize }
    }

    fn deallocation(&self) -> *mut usize {
        panic!("Not used on unix");
    }
}

pub struct StackPool {
    stacks: SegQueue<MmapStack>,
}

impl StackPool {
    pub fn new() -> Self {
        Self {
            stacks: Default::default(),
        }
    }

    pub fn get_stack(&self) -> MmapStack {
        if let Some(stack) = self.stacks.pop() {
            log::trace!("Reusing stack");
            stack
        } else {
            log::trace!("Creating new stack");
            match MmapStack::new() {
                Ok(stack) => stack,
                Err(err) => {
                    panic!("Failed to create task stack: {err}");
                }
            }
        }
    }

    pub fn store_stack(&self, stack: MmapStack) {
        self.stacks.push(stack);
    }
}
