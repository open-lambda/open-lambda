use open_lambda::json::{json, Number, Value};
use open_lambda::{info, error};

#[no_mangle]
fn f() {
    let mut args = if let Some(a) = open_lambda::get_args() {
        a
    } else {
        error!("No argument given");
        return;
    };

    let mut data_vec = Vec::new();
    let args = if let Some(a) = args.as_array_mut() {
        a
    } else {
        error!("Argument is not a vector");
        return;
    };

    for v in args.drain(..) {
        if let Some(f) = v.as_f64() {
            data_vec.push(f);
        } else {
            error!("Value argument vector is not a number");
            return;
        }
    }

    let vec: ndarray::Array1<f64> = data_vec.into();

    let result = Number::from_f64(vec.sum()).unwrap();
    let result = Value::Number(result);

    info!("Computed {} -> {}", vec, result);

    let j = json!({
        "result": result
    });

    if let Err(e) = open_lambda::set_result(&j) {
        error!("Failed to set result: {}", e);
    }
}
