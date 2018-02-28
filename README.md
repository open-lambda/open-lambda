# OpenLambda

OpenLambda is an Apache-licensed serverless computing project, written
in Go and based on Linux containers.  The primary goal of OpenLambda
is to enable exploration of new approaches to serverless computing.  Our
research agenda is described in more detail in a [HotCloud '16
paper](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf).

## Getting Started

OpenLambda relies heavily on operations that require root privilege. To
simplify this, we suggest that you run all commands as the root user 
(i.e., run `sudo -s` before building or running OpenLambda). Additionally,
OpenLambda is only actively tested on Ubuntu 14.04 & 16.04.

### Install Dependencies

First, run the dependency script to install necessary packages (e.g., Golang, 
Docker, etc.)

```
./quickstart/deps.sh
```

Now, build the OpenLambda worker & its dependencies and run integration
tests. These tests will spin up clusters in various configurations and
invoke a few lambdas.

```
make test-all
```

If these pass, congratulations! You now have a working OpenLambda installation.

### Start a Test Cluster

To manage a cluster, we will use the `admin` tool. This tool manages state via
a `cluster` directory on the local file system. More details on this tool can
be found [below](#admin-tool).

First, we need to create a cluster. Ensure that the path to the `cluster` directory 
exists, and it does not.

```
./bin/admin new -cluster my-cluster
```

Now start a local worker process in your cluster:

```
./bin/admin workers -cluster=my-cluster
```

Confirm that the worker is listening and responding to requests:

```
./bin/admin status -cluster=my-cluster
```

This should return something similar to the following:

```
Worker Pings:
  http://localhost:8080/status => ready [200 OK]

Cluster containers:

```

The default configuration uses a local directory to store handler
code, so creating new lambda functions is as simple as writing files
to the `./my-cluster/registry` directory.

Copy an example handler (`hello`) to this directory:

```
cp -r ./quickstart/handlers/hello ./my-cluster/registry/hello
```

Now send a request for the `hello` lambda to the worker via `curl`.
Handlers are passed a Python dictionary corresponding to the JSON
body of the request. `hello` will echo the `"name"` field of the
payload to show this.

```
curl -X POST localhost:8080/runLambda/hello -d '{"name": "Alice"}'
```

The request should return the following:

```
"Hello, Alice!"
```

To create your own lambda named `<NAME>`, write your code in
`./my-cluster/registry/<NAME>/lambda_func.py`, then invoke it via curl:

```
curl -X POST localhost:8080/runLambda/<NAME> -d '<JSON_STRING>'
```

Now, kill the worker process and (optionally) remove the `cluster`
directory.

```
./bin/admin kill -cluster=my-cluster
rm -r my-cluster
```

## Admin Tool

The `admin` tool is used to manage OpenLambda clusters. This tool manages 
state via a `cluster` directory on the local file system. Note that only
a single OpenLambda worker per machine is currently supported.

### Admin Tool Commands

TODO(Tyler): document the admin tool commands.

### Cluster Directory

TODO(Tyler): document the layout of the cluster directory.

## Configuration

TODO(Tyler): document the configuration parameters and how they interact.
Also describe how to use the packages.txt file in a handler directory to
specify dependencies.

## Architecture

TODO(Ed): concise description of the architecture.

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.
