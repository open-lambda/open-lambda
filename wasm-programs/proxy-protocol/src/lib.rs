use open_lambda_protocol::{CollectionId, Operation, OpResult};
use schema::ValueType;
use serde::{Serialize, Deserialize};

#[ derive(Serialize, Deserialize) ]
pub enum ProxyMessage {
    GetSchema{ collection: String },
    SchemaResult{ identifier: CollectionId, key: ValueType, fields: Vec<(String, ValueType)>},
    ExecuteOperation{ collection: CollectionId, op: Operation },
    OperationResult{ result: OpResult },
    TxCommitRequest,
    TxCommitResult{ result: bool },
}
