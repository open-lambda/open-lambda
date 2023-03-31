use open_lambda::{function_call, json};

#[open_lambda_macros::main_func]
fn main() {
    function_call("noop", &json::Value::Null).expect("Function call failed");
}
