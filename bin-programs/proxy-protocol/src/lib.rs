use serde::{Serialize, Deserialize};

#[ derive(Serialize, Deserialize) ]
pub enum ProxyMessage {
    CallRequest{ func_name: String, arg_string: String },
    CallResult(Result<String, String>),
}
