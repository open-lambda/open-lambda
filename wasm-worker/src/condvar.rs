/// Condvar wrapper for tokio

use std::sync::atomic;
use tokio::sync::{Mutex, MutexGuard, Notify};

pub struct Condvar {
    notifier: Notify,
    /// Atomic here is only used to make Convdar Send and Sync
    /// Callers still need to hold the associated mutex to avoid data races
    wait_count: atomic::AtomicU32,
}

impl Condvar {
    pub fn new() -> Self {
        Self{ wait_count: atomic::AtomicU32::new(0), notifier: Notify::new() }
    }

    pub async fn wait<'a, 'b, T>(&self, lock: MutexGuard<'a, T>, mutex: &'b Mutex<T>) -> MutexGuard<'b, T> {
        self.wait_count.fetch_add(1, atomic::Ordering::SeqCst);
        drop(lock);

        self.notifier.notified().await;
        mutex.lock().await
    }

    pub fn notify_one<'a, T>(&self, lock: MutexGuard<'a, T>) {
        let count = self.wait_count.load(atomic::Ordering::SeqCst);

        if count > 0 {
            self.wait_count.store(count-1, atomic::Ordering::SeqCst);
            drop(lock);

            self.notifier.notify_one();
        }
    }

    pub fn notify_all<'a, T>(&self, lock: MutexGuard<'a, T>) {
        let count = self.wait_count.swap(0, atomic::Ordering::SeqCst);
        drop(lock);

        for _ in 0..count {
            self.notifier.notify_one();
        }
    }
}
