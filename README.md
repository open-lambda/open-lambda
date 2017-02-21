# OpenLambda

OpenLambda is an Apache-licensed serverless computing project, written
in Go and based on Linux containers.  One of the goals of OpenLambda
is to enable exploration of new approaches to serverless computing.  Our
research agenda is described in more detail in a [HotCloud '16
paper](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf)

## Getting Started

These instructions assume you are the root user.

Install Docker and the Go compiler, then build OpenLambda:

```
make
```

Now you can use the admin tool to create a local OpenLamda cluster:

```
./bin/admin new -cluster my-cluster
```

Now start a single local worker process in your cluster:

```
./bin/admin workers -cluster=my-cluster
```

Confirm that the worker is listening and responding to requests:

```
./bin/admin status -cluster=my-cluster
```

You should see something like the following:

```
Worker Pings:
  http://localhost:8080/status => ready [200 OK]

Cluster containers:

```

The default configuration uses a local directory to store handler
code, so creating new Lambda functions is as simple as writing files
in the ./my-cluster/registry.

Copy an example handler to this directory:

```
cp -r ./quickstart/handlers/hello ./my-cluster/registry/hello
```

AJAX is an RPC protocol often used by web applications to communicate
with backend services.  AJAX issues calls via HTTP, and marshalls
arguments using JSON.  Thus, we can issue an AJAX call from the
command line, via curl:

```
curl -X POST localhost:8080/runLambda/hello -d '{"name": "Alice"}'
```

The request should return the following:

```
"Hello, Alice!"
```

To create your own Lambda function named `<NAME>`, write your code to
`./my-cluster/registry/<NAME>/lambda_func.py`, then invoke it via curl:

```
curl -X POST localhost:8080/runLambda/<NAME> -d '<JSON>'
```

The `<JSON>` string will be parsed to a Python object and passed to
the `handler` function via the `event` argument.

## Running the tests

To run the unit tests:

```
make test
```

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.
