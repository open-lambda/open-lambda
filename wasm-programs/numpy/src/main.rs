use open_lambda::json::{json, Number, Value};
use open_lambda::{debug, error, fatal};

fn parse_array(mut jvalue: Value) -> Result<(Vec<usize>, Vec<f64>), ()> {
    if let Some(jvec) = jvalue.as_array_mut() {
        if jvec.is_empty() {
            error!("Array is empty");
            return Err(());
        }

        if jvec[0].is_number() {
            let mut vec = Vec::new();
            for field in jvec.drain(..) {
                if let Some(f) = field.as_f64() {
                    vec.push(f);
                } else {
                    error!("Some fields in the vector are not numeric");
                    return Err(());
                };
            }

            Ok((vec![vec.len()], vec))
        } else {
            let mut shape = None;
            let mut result = Vec::new();
            let num_children = jvec.len();

            for field in jvec.drain(..) {
                match parse_array(field) {
                    Ok((cshape, mut vector)) => {
                        if shape.is_none() {
                            shape = Some(cshape);
                        } else if shape.as_ref().unwrap() != &cshape {
                            error!("Shapes don't match");
                            return Err(());
                        }

                        result.append(&mut vector);
                    }
                    Err(()) => return Err(())
                }
            }

            let mut new_shape = vec![num_children];
            new_shape.append(&mut shape.unwrap());

            Ok((new_shape, result))
        }
    } else {
        error!("Argument is not a vector");
        Err(())
    }
}

#[ open_lambda_macros::main_func ]
fn main() {
    let args = if let Some(a) = open_lambda::get_args() {
        a
    } else {
        fatal!("No argument given");
    };

    debug!("Argument is `{}`", args);

    let (shape, vec) = if let Ok((shape, vec)) = parse_array(args) {
        (shape, vec)
    } else {
        fatal!("Failed to parse argument; is it a valid tensor?");
    };

    let din = ndarray::IxDyn(&shape);
    let tensor = ndarray::ArrayView::from_shape(din, &vec[..]).unwrap();

    let result = Number::from_f64(tensor.sum()).unwrap();
    let result = Value::Number(result);
    debug!("Computed {} -> {}", tensor, result);

    let j = json!({
        "result": result
    });

    if let Err(e) = open_lambda::set_result(&j) {
        error!("Failed to set result: {}", e);
    }
}
