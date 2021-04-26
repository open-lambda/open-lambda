use open_lambda::{get_args, get_collection, entry, fatal};

#[ open_lambda_macros::main_func ]
fn main() {
    let args = get_args().expect("No argument given");
    let num_gets;
    let num_puts;
    let num_deletes;
    let entry_size;

    if let Some(args) = args.as_object() {
        num_puts= args.get("num_puts").expect("Could not find `num_puts` argument")
            .as_i64().unwrap() as usize;
        num_gets = args.get("num_gets").expect("Could not find `num_gets` argument")
            .as_i64().unwrap() as usize;
        num_deletes = args.get("num_deletes").expect("Could not find `num_deletes` argument")
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

    for i in 0..num_puts {
        let mut data = Vec::new();
        data.resize(entry_size, 0);
        let string = String::from_utf8(data).unwrap();

        if let Err(e) = col.put(format!("key{}", i), entry!{"value" => string}) {
            open_lambda::fatal!("{}", e);
        }
    }

    for i in 0..num_gets {
        let mut data = Vec::new();
        data.resize(entry_size, 0);
        let string = String::from_utf8(data).unwrap();

        let expected = entry!{"value" => string};

        match col.get(format!("key{}", i)) {
            Ok(res) => assert_eq!(res, expected),
            Err(e) => { open_lambda::fatal!("{}", e); }
        }
    }

    for i in 0..num_deletes {
        col.delete(format!("key{}", i)).unwrap();
    }
}
