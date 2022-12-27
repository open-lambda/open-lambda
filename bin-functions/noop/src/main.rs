use open_lambda::log;

#[open_lambda_macros::main_func]
fn main() {
    log::info!("Hello, world!");
}
