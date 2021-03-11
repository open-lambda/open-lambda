use open_lambda::info;

#[ cfg(not(target="wasm32")) ]
fn main() {
    f()
}


#[no_mangle]
fn f() {
    info!("Hello world");
}
