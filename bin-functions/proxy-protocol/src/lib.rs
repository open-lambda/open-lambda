use serde::{Deserialize, Serialize};
use serde_bytes::ByteBuf;

#[derive(Serialize, Deserialize)]
pub struct CallData {
    pub fn_name: String,
    pub args: ByteBuf,
}

pub type CallResult = Result<ByteBuf, String>;

#[derive(Serialize, Deserialize)]
pub enum ProxyMessage {
    CallRequest(CallData),
    CallResult(CallResult),
}
