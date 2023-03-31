use open_lambda::{get_args, json, set_result};

use serde::Deserialize;

#[derive(Deserialize)]
struct MultiplyArgs {
    left: i64,
    right: i64,
}

#[open_lambda_macros::main_func]
fn main() {
    let jargs = get_args().expect("No argument given");
    let args: MultiplyArgs = json::from_value(jargs).unwrap();

    let result = args.left * args.right;

    set_result(&json::json!({
        "result": result,
    }))
    .unwrap();
}
