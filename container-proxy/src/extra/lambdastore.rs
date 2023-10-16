use lambda_store_client::{Client, ObjectType, ObjectTypeId};
use open_lambda_proxy_protocol::CallResult;

use parking_lot::Mutex as PMutex;

use serde_bytes::ByteBuf;

use async_once_cell::OnceCell;

static ADDRESS: PMutex<Option<String>> = PMutex::new(None);
static CLIENT: OnceCell<Client> = OnceCell::new();

pub fn set_address(address: String) {
    log::debug!("Lambdastore address set to `{address}`");
    let _ = ADDRESS.lock().insert(address);
}

async fn get_or_create_client<'a>() -> &'a Client {
    CLIENT.get_or_init(async {
        let address = ADDRESS.lock().as_ref().unwrap().clone();
        match  lambda_store_client::create_client(&address).await {
            Ok(conn) => conn,
            Err(err) => {
                panic!("Failed to connect to lambdastore @{address}: {err}");
            }
        }
    }).await
}

pub async fn call(func_name: &str, args: &[u8]) -> CallResult {
    let client = get_or_create_client().await;

    if func_name == "create_object" {
        let (app_name, typename) = bincode::deserialize(args).unwrap();
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
        let app_name: String = bincode::deserialize(args).unwrap();
        let object_types: Vec<(ObjectTypeId, String, ObjectType)> = client
            .get_object_types_by_app_name(&app_name)
            .unwrap()
            .into_iter()
            .map(|(id, name, info)| (id, name, (*info).clone()))
            .collect();

        let data = bincode::serialize(&object_types).unwrap();
        Ok(ByteBuf::from(data))
    } else {
        panic!("Got unexpected lambdastore function call: {func_name}");
    }
}
