pub use log::{debug, error, info};

#[macro_export]
macro_rules! fatal {
    ($($args:tt)*) => {
        $crate::log::error!("Got fatal error: {}", format!($($args)*));
        panic!($($args)*);
    }
}

pub use fatal;
