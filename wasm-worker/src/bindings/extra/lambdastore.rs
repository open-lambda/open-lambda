use lambda_store_client::Client;
use open_lambda_proxy_protocol::CallResult;

static mut CLIENT: Option<Client> = None;

/// SAFETY: Only call this during startup
pub async unsafe fn create_client(address: &str) -> Result<(), String> {
    CLIENT = Some(lambda_store_client::create_client(address).await?);
    Ok(())
}

pub async fn call(func_name: &str, args: &[u8]) -> CallResult {
    if func_name == "execute_operation" {
        todo!();
    } else {
        panic!("Got unexpected lambdastore function call: {func_name}");
    }
}
