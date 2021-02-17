use open_lambda::json::{json, Number, Value};

#[no_mangle]
fn f() {
    let mut args = open_lambda::get_args().expect("No argument given");

    let mut data_vec = Vec::new();
    for v in args.as_array_mut().expect("Argument is not an vector").drain(..) {
        if let Some(f) = v.as_f64() {
            data_vec.push(f);
        } else {
            panic!("Value argument vector is not a number");
        }
    }

    let vec: ndarray::Array1<f64> = data_vec.into();
    let result = Number::from_f64(vec.sum()).unwrap();
    let result = Value::Number(result);

    open_lambda::set_result(json!({
        "result": result
    }))
}
