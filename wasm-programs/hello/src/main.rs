use open_lambda::info;

fn main() {
    #[ cfg(not(target_arch="wasm32")) ]
    f()
}

#[no_mangle]
fn f() {
    open_lambda::init();
    info!("Hello world");
}
