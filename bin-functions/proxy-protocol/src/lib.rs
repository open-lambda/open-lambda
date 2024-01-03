use serde::{Deserialize, Serialize};
use serde_bytes::ByteBuf;

#[derive(Debug, Serialize, Deserialize)]
pub struct FuncCallData {
    pub fn_name: String,
    pub args: ByteBuf,
}

pub type CallResult = Result<ByteBuf, String>;

#[derive(Debug, Serialize, Deserialize)]
pub enum ProxyMessage {
    FuncCallRequest(FuncCallData),
    FuncCallResult(CallResult),
}
