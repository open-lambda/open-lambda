use serde_json as json;

use crate::internal::ipc as internal_ipc;
use open_lambda_proxy_protocol::CallResult;

pub fn host_call(
    namespace: &str,
    func_name: &str,
    args: &json::Value,
) -> Result<Option<json::Value>, String> {
    let arg_string = serde_json::to_string(args).unwrap();
    let arg_data = arg_string.into_bytes();

    let result = internal_ipc::host_call(namespace, func_name, arg_data);
    parse_json_result(result)
}

pub fn function_call(func_name: &str, args: &json::Value) -> Result<Option<json::Value>, String> {
    let arg_string = serde_json::to_string(args).unwrap();
    let arg_data = arg_string.into_bytes();

    let result = internal_ipc::function_call(func_name, arg_data);
    parse_json_result(result)
}

fn parse_json_result(result: CallResult) -> Result<Option<json::Value>, String> {
    match result {
        Ok(result_data) => {
            if result_data.is_empty() {
                // Is this needed? Result should always be a valid json
                Ok(None)
            } else {
                let result_string = String::from_utf8(result_data.into_vec())
                    .expect("Result not a valid utf-8 string?");

                match json::from_str(&result_string) {
                    Ok(json) => Ok(Some(json)),
                    Err(err) => Err(format!(
                        "Failed to parse call result JSON \"{result_string}\": {err}"
                    )),
                }
            }
        }
        Err(msg) => Err(msg),
    }
}
