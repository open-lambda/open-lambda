use open_lambda::{OpError, get_args, get_collection, entry, fatal};

#[ open_lambda_macros::main_func ]
fn main() {
    let args = get_args().expect("No argument given");
    let num_entries;
    let entry_size;

    if let Some(args) = args.as_object() {
        num_entries = args.get("num_entries").expect("Could not find `num_entries` argument")
            .as_i64().unwrap() as usize;
        entry_size = args.get("entry_size").expect("Coult not find `entry_size` argument")
            .as_i64().unwrap() as usize;
    } else {
        fatal!("No arguments given");
    }

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
        let mut data = Vec::new();
        data.resize(entry_size, 0);
        let string = String::from_utf8(data).unwrap();

        if let Err(e) = col.put(format!("key{}", i), entry!{"value" => string}) {
            open_lambda::fatal!("{}", e);
        }
    }

    for i in 0..num_entries {
        let mut data = Vec::new();
        data.resize(entry_size, 0);
        let string = String::from_utf8(data).unwrap();

        let expected = entry!{"value" => string};

        match col.get(format!("key{}", i)) {
            Ok(res) => assert_eq!(res, expected),
            Err(e) => { open_lambda::fatal!("{}", e); }
        }
    }

    for i in 0..num_entries {
        col.delete(format!("key{}", i)).unwrap();
    }
}
