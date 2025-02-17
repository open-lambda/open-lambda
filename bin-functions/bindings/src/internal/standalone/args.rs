pub use serde_json as json;
use std::time::{SystemTime, UNIX_EPOCH};

pub fn get_args() -> Option<json::Value> {
    let mut args = std::env::args();

    args.next().unwrap();

    if let Some(arg) = args.next() {
        let jvalue = json::from_str(&arg).expect("Failed to parse JSON");
        Some(jvalue)
    } else {
        None
    }
}

pub fn set_result(value: &json::Value) -> Result<(), json::Error> {
    use std::fs::File;
    use std::io::Write;

    let jstr = json::to_string(value)?;

    let path = "/tmp/output";

    let mut file = File::create(path).unwrap();
    file.write_all(jstr.as_bytes()).unwrap();

    file.sync_all().expect("Writing to disk failed");
    log::debug!("Created output file at {path}");

    Ok(())
}

pub fn get_unix_time() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("System time before UNIX epoch")
        .as_secs()
}
