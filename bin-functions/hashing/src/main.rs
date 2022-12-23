use open_lambda::get_args;

use rand::rngs::SmallRng;
use rand::{Fill, SeedableRng};

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

    let mut input_vec = Vec::<u8>::new();
    input_vec.resize(input_len, 0);

    let mut rng = SmallRng::from_entropy();
    input_vec.try_fill(&mut rng).unwrap();

    for _ in 0..num_hashes {
        let mut hasher = Sha512::new();
        hasher.update(&input_vec);

        let _ = hasher.finalize();
    }
}
