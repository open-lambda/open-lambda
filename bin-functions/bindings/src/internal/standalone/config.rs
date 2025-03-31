use std::collections::HashMap;
use std::fs::read_to_string;

use lazy_static::lazy_static;

fn parse_config_file() -> Result<HashMap<String, String>, String> {
    let data = read_to_string("/config.toml").map_err(|err| format!("{err}"))?;
    toml::from_str(&data).map_err(|err| format!("{err}"))
}

pub fn get_config_value(key: &str) -> Result<String, String> {
    lazy_static! {
        static ref CONFIG: Result<HashMap<String, String>, String> = parse_config_file();
    };

    match CONFIG.as_ref() {
        Ok(config) => {
            if let Some(val) = config.get(key) {
                Ok(val.clone())
            } else {
                Err(format!("No such config entry: {key}"))
            }
        }
        Err(err) => Err(err.clone()),
    }
}
