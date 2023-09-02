use lambda_store_client::{Client, ObjectType, ObjectTypeId};
use open_lambda_proxy_protocol::CallResult;

use serde_bytes::ByteBuf;

static mut CLIENT: Option<Client> = None;

/// SAFETY: Only call this during startup
pub async unsafe fn create_client(address: &str) -> Result<(), String> {
    CLIENT = Some(lambda_store_client::create_client(address).await?);
    Ok(())
}

pub async fn call(func_name: &str, args: &[u8]) -> CallResult {
    // SAFETY: lambdastore is always initialized at this point
    // otherwise the wasm-worker will not start successfully
    let client = unsafe { CLIENT.as_ref().expect("lambdastore not initialized") };

    if func_name == "create_object" {
        let (app_name,typename) = bincode::deserialize(args).unwrap();
        match client.create_object(app_name, typename).await {
            Ok(object) => {
                let object_id = bincode::serialize(&object.get_identifier()).unwrap();
                Ok(ByteBuf::from(object_id))
            }
            Err(err) => Err(format!("{err}")),
        }
    } else if func_name == "execute_operation" {
        let result = loop {
            let data_op = bincode::deserialize(args).unwrap();
            let mut tx = client.begin_transaction();
            let result = tx.execute_operation(data_op).await;
            if tx.commit().await.is_ok() {
                break result;
            }
        };

        result.map_err(|err| format!("{err}"))
    } else if func_name == "execute_range_query" {
        let result = loop {
            let range_op = bincode::deserialize(args).unwrap();
            let mut tx = client.begin_transaction();
            let result = tx.execute_range_query(range_op).await;
            if result.is_ok() && tx.commit().await.is_ok() {
                break result;
            }
        };
        result.map_err(|err| format!("{err}"))
    } else if func_name == "get_configuration" {
        let object_types: Vec<(ObjectTypeId, String, ObjectType)> = client
            .get_object_types()
            .into_iter()
            .map(|(id, name, info)| (id, name, (*info).clone()))
            .collect();

        let data = bincode::serialize(&object_types).unwrap();
        Ok(ByteBuf::from(data))
    } else {
        panic!("Got unexpected lambdastore function call: {func_name}");
    }
}
