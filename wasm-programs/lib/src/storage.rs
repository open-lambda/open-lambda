#[ cfg(target_arch="wasm32") ]
use schema::Schema;

pub use schema::Value;

#[ cfg(target_arch="wasm32") ]
use std::collections::HashMap;

#[ cfg(target_arch="wasm32") ]
use open_lambda_protocol::{CollectionInfo, CollectionId, OpResult, Operation};

#[ cfg(not(target_arch="wasm32")) ]
static mut DATABASE: Option<lambda_store_client::Client> = None;

pub use open_lambda_protocol::OpError;

#[ macro_export ]
macro_rules! entry(
    { $($key:expr => $value:expr),+ } => {
        {
            let mut m = ::std::collections::HashMap::new();
            $(
                m.insert($key.to_string(), $value.into());
            )+
            m
        }
     };
);

#[ cfg(target_arch="wasm32") ]
mod internal {
    #[link(wasm_import_module="ol_storage")]
    extern "C" {
        pub fn execute_operation(op_data: *const u8, op_data_len: u32, out_len: *mut u64) -> i64;

        pub fn get_collection_schema(col_name_ptr: *const u8, col_name_len: u32, out_len: *mut u64) -> i64;
    }
}

#[ cfg(target_arch="wasm32") ]
pub struct Collection {
    identifier: CollectionId,
    name: String,
    schema: Schema
}

#[ cfg(target_arch="wasm32") ]
impl Collection {
    pub fn get<K: Into<Value>>(&self, key: K) -> Result<HashMap<String, Value>, OpError> {
        let key: Value = key.into();
        let key = key.serialize_inner();

        let operation = Operation::Get{
            key, collection: self.identifier
        };

        self.execute_operation(operation, None)
    }

    pub fn put<K: Into<Value>>(&self, key: K, mut fields: HashMap<String, Value>) -> Result<(), OpError> {
        let key: Value = key.into();
        let key = key.serialize_inner();

        let mut row = self.schema.build_entry();

        for (key, value) in fields.drain() {
            row = row.set_field_from_value(key, &value);
        }

        let operation = Operation::Put{
            key, collection: self.identifier, value: row.done()
        };

        let filter = vec![];

        match self.execute_operation(operation, Some(filter)) {
            Ok(_) => Ok(()),
            Err(e) => Err(e)
        }
    }

    fn execute_operation(&self, operation: Operation, filter: Option<Vec<String>>) -> Result<HashMap<String, Value>, OpError> {
        let op_data = bincode::serialize(&operation).unwrap();

        let mut out_len = 0u64;
        let data_ptr = unsafe{
            let len = op_data.len() as u32;
            let out_len_ptr = (&mut out_len) as *mut u64;
            internal::execute_operation(op_data.as_ptr(), len, out_len_ptr)
        };

        if data_ptr <= 0 {
            panic!("Got unexpected error");
        }

        let out_len = out_len as usize;
        let data = unsafe {
            Vec::<u8>::from_raw_parts(data_ptr as *mut u8, out_len, out_len)
        };

        let result: OpResult = bincode::deserialize(&data)
                        .expect("Failed to deserialize OpResult");

        match result {
            Ok(entry) => {
                let result = if let Some(f) = filter {
                    let filter: Vec<&str> = f.iter().map(|s| s as &str).collect();
                    self.schema.get_fields_with_filter(&entry, &filter)
                } else {
                    self.schema.get_fields(&entry)
                };

                match result {
                    Ok(f) => Ok(f),
                    Err(err) => Err(OpError::SchemaError(err))
                }
            },
            Err(err) => Err(err)
        }
    }

    pub fn get_name(&self) -> &str {
        &self.name
    }
}

#[ cfg(target_arch="wasm32") ]
pub fn get_collection<T: ToString>(name: T) -> Option<Collection> {
    let name = name.to_string();
    let mut out_len = 0u64;
    let schema_ptr = unsafe {
        internal::get_collection_schema(name.as_bytes().as_ptr(),  name.len() as u32, (&mut out_len) as *mut u64)
    };

    if schema_ptr < 0 {
        return None;
    }

    let out_len = out_len as usize;
    let data = unsafe {
        Vec::<u8>::from_raw_parts(schema_ptr as *mut u8, out_len, out_len)
    };

    let info: CollectionInfo = bincode::deserialize(&data).unwrap();
    let schema = Schema::from_parts(info.key_type, info.fields);

    Some( Collection{ name, identifier: info.identifier, schema } )
}

#[ cfg(not(target_arch="wasm32")) ]
pub fn get_collection<T: ToString>(name: T) -> Option<lambda_store_client::Collection> {
    let database = if let Some(inner) = unsafe{ DATABASE.take() } {
        inner
    } else {
        lambda_store_client::create_client("localhost")
    };

    let col = database.get_collection(name.to_string());

    unsafe{ DATABASE = Some(database) };
    col.ok()
}

