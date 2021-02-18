use open_lambda::json::{json, Number, Value};
use open_lambda::{debug, error};

fn parse_array(mut jvalue: Value) -> Result<(Vec<usize>, Vec<f64>), ()> {
    if let Some(jvec) = jvalue.as_array_mut() {
        if jvec.len() == 0 {
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
        return Err(());
    }
}

#[no_mangle]
fn f() {
    let args = if let Some(a) = open_lambda::get_args() {
        a
    } else {
        error!("No argument given");
        return;
    };

    let (shape, vec) = if let Ok((shape, vec)) = parse_array(args) {
        (shape, vec)
    } else {
        return;
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
