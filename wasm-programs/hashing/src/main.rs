use open_lambda::{fatal, get_args};

use sha2::{Digest, Sha512};


fn main() {
    #[ cfg(not(target_arch="wasm32")) ]
    f()
}

#[no_mangle]
fn f() {
    open_lambda::init();

    let args = get_args().expect("No argument given");
    let num_hashes;
    let input_len;

    if let Some(args) = args.as_object() {
        num_hashes = args.get("num_hashes").expect("Could not find `num_hashes` argument")
            .as_i64().unwrap() as usize;
        input_len = args.get("input_len").expect("Coult not find `input_len` argument")
            .as_i64().unwrap() as usize;
    } else {
        fatal!("Invalid argument");
    }

    let mut input_vec = Vec::<u8>::new();
    input_vec.resize(input_len, 0);

    /* rand not supported on WASM
    let mut rng = rand::thread_rng();
    input_vec.try_fill(&mut rng).unwrap();*/

    for _ in 0..num_hashes {
        let mut hasher = Sha512::new();
        hasher.update(&input_vec);

        let _ = hasher.finalize();
    }
}
