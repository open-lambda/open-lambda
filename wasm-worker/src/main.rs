use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Result, Server, StatusCode};

mod programs;
use programs::ProgramManager;

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let addr = "127.0.0.1:1337".parse().unwrap();

    let make_service =
        make_service_fn(|_| async { Ok::<_, hyper::Error>(service_fn(response_examples)) });

    let server = Server::bind(&addr).serve(make_service);

    println!("Listening on http://{}", addr);

    if let Err(e) = server.await {
        eprintln!("server error: {}", e);
    }
}

fn execute_function(name: &str) -> Result<Response<Body>> {
    let response = Response::builder()
        .status(StatusCode::OK)
        .body(Body::empty())
        .unwrap();

    Ok(response)
}

async fn response_examples(req: Request<Body>) -> Result<Response<Body>> {
    if let &Method::POST = req.method() {
        let path = req.uri().path().split("/").collect::<Vec<&str>>();

        if path.len() != 2 || path[0] != "run" {
            panic!("Not a run request");
        }

        execute_function(path[1])
    } else {
        panic!("not a POST request");
    }
}
