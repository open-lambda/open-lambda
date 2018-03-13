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

The simplest admin command, `worker-exec`, allows you to launch a
foreground OpenLambda process.  For example:

```
  admin worker-exec --config=worker.json
```

The above command starts running a single worker with a configuration
specified in the worker.json file (described in detail later).  All
log output goes to the terminal (i.e., stdout), and you can stop the
process with ctrl-C.

Suppose worker.json contains the following line:

```
  "worker_port": "8080"
```

While the process is running, you may ping it from another terminal
with the following command:

```
  curl http://localhost:8080/status
```

If the worker is ready, the status request will return a "ready"
message.

Of course, you will typically want to run one (or maybe more) workers
as servers in the background on your machine.  Most of the remaining
admin commands allow you to manage these long-running workers.

An OpenLambda worker requires a local file-system location to store
handler code, logs, and various other data.  Thus, when starting a new
local cluster, the first step is to indicate where the cluster data
should reside with the `new` command:

```
   admin new --cluster=<ROOT>
```

For OpenLambda, a local cluster's name is the same as the file
location.  Thus, <ROOT> should refer to a local directory that will be
created for all OpenLambda files.  The layout of these files in the
<ROOT> directory is described in detail below.  You will need to pass
the cluster name/location to all future admin commands that manage the
cluster.

The "<ROOT>/config/template.json" file in the cluster located at
"<ROOT>" will contain many configuration options specified as
keys/values in JSON.  These setting will be used for every new
OpenLambda worker.  You can modify these values by specifying override
values (again in JSON) using the `setconf` command.  For example:

```
  ./admin setconf --cluster=<ROOT> '{"sandbox": "sock", "registry": "local"}'
```

In the above example, the configuration is modified so that workers
will use the local registry and the "sock" sandboxing engine.

Once configuration is complete, you can launch a specified number of
workers (currently only one is supported?) using the following
command:

```
  ./admin workers --cluster=<NAME> --num-workers=<NUM> --port=<PORT>
```

This will create a specified number of workers listening on ports
starting at the given value.  For example, suppose <NUM>=3 and
<PORT>=8080.  The `workers` command will create three workers
listening on ports 8080, 8081, and 8082.  The `workers` command is
basically a convenience wrapper around the `worker-exec` command.  The
`workers` command does three things for you: (1) creates a config file
for each worker, based on template.json, (2) invokes `worker-exec` for
each requested worker instance, and (3) makes the workers run in the
background so they continue executing even if you exit the terminal.

When you want to stop a local OpenLambda cluster, you can do so by
executing the of `kill` command:

```
  ./admin kill --cluster=<NAME>
```

This will halt any processes or containers associated with the
cluster.

In addition to the above commands for managing OpenLambda workers, two
admin commands are also available for managing an OpenLambda handler
store.  First, you may launch the OpenLambda registry with the
following `registry` command:

```
   ./admin registry --port=<PORT> --access-key=<KEY> --secret-key=<SECRET>
```

The registry will start listening on the designated port.  You may
generate the KEY and SECRET randomly yourself if you wish (or you may
use some other hard-to-guess SECRET).  Keep these values handy for
later uploading handlers.

The "<ROOT>/config/template.json" file specifies registry mode and
various registry options.  You may manually set these, but as a
convenience, the `registry` command will automatically
populate the configuration file for you when you launch the registry
process.  Thus, to avoid manual misconfiguration, we recommend running
`./admin registry` before running `./admin workers`.  Or, if you wish
to use the local-directory mode for your registry, simply never run
`./admin registry` (the default configs use local-directory mode).

After the registry is running, you may upload handlers to it via the
following command:

```
  ./admin upload --cluster=<NAME> --handler=<HANDLER-NAME> --file=<TAR> --access-key=<KEY> --secret-key=<SECRET>
```

The above command should use the KEY/SECRET pair used when you
launched the registry earlier.  The <TAR> can refer to a handler
bundle.  This is just a .tar.gz containing (at a minimum) a
lambda_func.py file (for the code) and a packages.txt file (to specify
the Python dependencies).

### Writing Handlers

TODO(Tyler): describe how to write and upload handlers

### Cluster Directory

Suppose you just ran the following:

```
admin new --cluster=./my-cluster
```

You'll find six subdirectories in the `my-cluster' directory:
`config`, `logs`, `base`, `packages`, `registry`, and
`workers`.

The config directory will contain, at a minimum, a `template.json`
file.  Once you start workers, each worker will have an additional
config file in this directory named `worker-<N>.json` (the admin tool
creates these by copying first copying `template.json`, then
populating additional fields specific to the worker).

Each running worker will create two files in the `logs` directory:
`worker-<N>.out` and `worker-<N>.pid`.  The ".out" files contain the
log output of the workers; this is a good place to start if the
workers are not reachable or if they are returning unexpected errors.
The ".pid" files each contain a single number representing the process
ID of the corresponding worker process; this is mostly useful to the
admin kill tool for identifying processes to halt.

All OpenLambda handlers run on the same base image, which is dumped via
Docker into the `my-cluster/base` directory.  This contains a standard Ubuntu
image with additional OpenLambda-specific components. This base is
accessed on a read-only basis by every handler when using SOCK containers.

The `./my-cluster/packages` directory is mapped (via a read-only bind mount) 
into all containers started by the worker, and contains all of the PyPI 
packages installed to workers on this machine.

As discussed earlier, OpenLambda can use a separate registry service
to store handlers, or it can store them in a local directory; the
latter is more convenient for development and testing.  Unless
configured otherwise, OpenLambda will treat the
`./my-cluster/registry` directory as a handler store.  Creating a
handler named "X" is as simple as creating a directory named
`./my-cluster/registry/X` and writing your code therein.  No
compression is necessary in this mode; the handler code for "X" can be
saved here: `./my-cluster/registry/X/lambda_func.py`.

Each worker has its own directory for various state.  The storage for
worker N is rooted at `./my-cluster/workers/worker-<N>`.  Within that
directory, handler containers will have scratch space at
`./handlers/<handler-name>/<instance-number>`.  For example, all
containers created to service invocations of the "echo" handler will
have scratch space directories inside the `./handlers/echo` directory.
Additionally, there is a directory `./my-cluster/workers/worker-<N>/import-cache`
that contains the communication directory mapped into each import cache
entry container.

Suppose there is an instance of the "echo" handler with ID "3".  That
container will have it's scratch space at `./handlers/echo/3` (within
the worker root, `./my-cluster/workers/worker-<N>`).  The handler may
write temporary files in that directory as necessary.  In addition to
these, there will be three files: `server_pipe` sock file (used by the
worker process to communicate with the handler) and `stdout` and
`stderr` files (handler output is redirected here).  When debugging a
handler, checking these output files can be quite useful.

Note that the same directory can appear at different locations in the
host and in a guest container.  For example, containers for two
handlers named "function-A" and "function-B" might have scratch space
on the host allocated at the following two locations:

```
./my-cluster/workers/worker-0/handlers/function-A/123
./my-cluster/workers/worker-0/handlers/function-B/321
```

As a developer debugging the functions, you may want to peek in the
above directories to look for handler output and generated files.
However, in order to write code for a handler that generates output in
the above locations, you will need to write files to the `/host`
directory (regardless of whether you're writing code for function-A or
function-B) because that is where scratch space is always mapped
within a lambda container.

## Configuration

TODO(Tyler): document the configuration parameters and how they interact.
Also describe how to use the packages.txt file in a handler directory to
specify dependencies.

## Architecture

TODO(Ed): concise description of the architecture.

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.
