use open_lambda::{get_args, rng};

use rand::TryRngCore;

use sha2::{Digest, Sha512};

#[open_lambda_macros::main_func]
fn main() {
    let args = get_args().expect("No argument given");

    let args = args.as_object().expect("invalid argument");
    let num_hashes = args
        .get("num_hashes")
        .expect("Could not find `num_hashes` argument")
        .as_i64()
        .unwrap() as usize;
    let input_len = args
        .get("input_len")
        .expect("Coult not find `input_len` argument")
        .as_i64()
        .unwrap() as usize;

    let mut input_vec = vec![0; input_len];
    rng().try_fill_bytes(&mut input_vec).unwrap();

    for _ in 0..num_hashes {
        let mut hasher = Sha512::new();
        hasher.update(&input_vec);

        let _ = hasher.finalize();
    }
}
