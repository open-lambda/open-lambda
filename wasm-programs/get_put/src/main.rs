use open_lambda::{OpError, get_collection, entry};

fn main() {
    #[ cfg(not(target_arch="wasm32")) ]
    f()
}

#[no_mangle]
fn f() {
    open_lambda::init();
    let num_entries = 1_000;

    let col = match get_collection("default") {
        Some(col) => col,
        None => open_lambda::fatal!("no such collection")
    };

    for i in 0..num_entries {
        if col.get(format!("key{}", i)) != Err(OpError::NoSuchEntry) {
            open_lambda::fatal!("Entry should not exist yet");
        }
    }

    for i in 0..num_entries {
        if let Err(e) = col.put(format!("key{}", i), entry!{"value" => 5000}) {
            open_lambda::fatal!("{}", e);
        }
    }

    for i in 0..num_entries {
        let expected = entry!{"value" => 5000};

        match col.get(format!("key{}", i)) {
            Ok(res) => assert_eq!(res, expected),
            Err(e) => { open_lambda::fatal!("{}", e); }
        }
    }

    for i in 0..num_entries {
        col.delete(format!("key{}", i)).unwrap();
    }
}
