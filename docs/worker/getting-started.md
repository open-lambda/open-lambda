# Deploying an OpenLambda Worker

## Dependencies

OpenLambda relies heavily on operations that require root privilege.
To simplify this, we suggest that you run all commands as the root user.

OpenLambda is only actively tested on Ubuntu 24.04 LTS.  If you try
running on a different distro, one thing you'll definitely need is
cgroups2 mounted at /sys/fs/cgroup (`mount | grep cgroup2`).
OpenLambda does not work with cgroups v1.

Make sure you have all basic dependencies installed:
```
apt update
apt install -y docker.io llvm-14-dev libclang-common-14-dev build-essential python3 zlib1g-dev golang-go
```

If the `go version` is 1.21+ (as it should be on the Ubuntu 24.04), a
build will automatically pull the Go version specified in
`./go/go.mod` for the sake of building OpenLambda.

### Optional: Full Deployment (with WebAssembly Support)

OpenLambda has support for both Python-based and WebAssemby-based
lambdas (e.g., you could create a lambda by compiling Rust to
WebAssembly).

You can skip this part if you only want Python (a "min" deployment).

If want to write Rust-based lambdas, you need to have a recent nightly
version of Rust, the wasm32 toolchain, and the `cross` tool
installed. The easiest way to do this is:

```
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain=nightly-2025-02-15
source $HOME/.cargo/env
rustup target add wasm32-unknown-unknown
cargo install cross
```

Finally, add your user to the docker group to enable cross-compilation
of native binaries to OpenLambda's environment. Do not forget to
restart your shell/session afterwards!

```
sudo gpasswd -a $USER docker
```

## Build

### Python Only

Just run this:

```
make ol imgs/ol-min
```

### Python+WASM (for Rust support)

```
make all
make sudo-install
```

The `make sudo-install` step installs the binaries (`ol`, `ol-wasm`, and `ol-container-proxy`) to `/usr/local/bin/`, which is required for running the full test suite.

If you have a complete setup (Python+WASM), now is a good time to test
your environment: `make test-all`.

## Init

Run `./ol worker --help` to see an overview of the most important
worker commands (init/up/down/status).

The `init` command creates a new directory (named "default-ol" by
default) with a config file for the worker ("config.json"), a base
image used (read only) by all lambda instances ("lambda"), and other
resources.

The first init might take a couple minutes because it needs to
populate the "lambda" directory by extracting a Docker image created
during the build step (it will be 1-3 GB, depending on whether chose a
"min" or complete setup).

If you want the complete setup, just run this:

```
./ol worker init
```

The `default-ol/lambda` will then contain a dump of the `ol-wasm`
Docker image.

You read about other options with `./ol worker init --help`.  The
directory location and base image are configurable.  For example, you
could deploy the Python-only image to a directory called "myworker" like this:

```
./ol worker init -p myworker -i ol-min
```

If you want to customize the base image, you can create your own
Docker image from `ol-min` or `ol-wasm` and pass the image name to
`init`.

## Run Worker

Default config settings were saved to "./default-ol/config.json"
during the `init` command (or a similar location if you chose a
different path with `-p`).  The defaults attempt to set reasonable
limits based on the memory of your system.

You can optionally modify "config.json" before launching the worker like this:

```
./ol worker up
```

Note that if you passed a different worker directory to `init` (like
`-p myworker`), you'll need to pass the same to the `up` command.

You can cleanly kill the worker anytime with `ctrl-C`.

### Detached Mode

If you want to run the worker in detached mode (i.e., in the
background), just start it again with the `-d` flag:

```
./ol worker up -d
```

In detached mode, you can check the worker status with `./ol worker
status` or stop it cleanly with `./ol worker down`.

After initializing a worker directory once (`init` command), you can
start and stop a worker many times without reinitializing.  You can
change the config file, but the changes won't take effect until you
restart the worker.

## Using Provided Examples

OpenLambda includes a variety of example lambda functions in the `examples/` directory to help you get started quickly. These examples demonstrate different use cases and features:

- `examples/echo/` - Simple echo function that returns the input
- `examples/hello/` - Basic hello world function  
- `examples/numpy21/`, `examples/numpy22/` - Examples using NumPy
- `examples/pandas/` - Data processing with Pandas
- `examples/flask-test/` - Web framework integration
- `examples/timeout/` - Function with timeout configuration
- And many more...

To use any of these examples, simply install them directly:

```bash
./ol admin install examples/echo/
./ol admin install examples/hello/
./ol admin install examples/numpy21/
```

Then invoke them with curl:

```bash
curl -X POST localhost:5000/run/echo -d '{"hello": "world"}'
```

## Creating and Installing a Custom Lambda

You can also create your own lambda functions. Create the function directory:

```bash
mkdir -p echo
```

Now, create a file named `f.py` inside the `echo` directory with the following content:

```python
def f(event):
    return event
```

With the worker running, you can install the lambda function using the `ol admin install` command:

```bash
./ol admin install echo/
```

This command will package the `echo` directory into a `.tar.gz` file and upload it to the worker's registry.

If you initialized a worker with a specific path (e.g., `./ol worker init -p myworker`), you must specify the same path when installing a lambda.

```bash
./ol admin install -p myworker echo/
```

If no `-p` flag is specified, the command will default to the worker running on port 5000 using the default config.

### Installing from a Git Repository

You can also install lambdas directly from a Git repository (GitHub, GitLab, etc.):

```bash
./ol admin install https://github.com/open-lambda/hello-lambda-example.git
```

This works with both HTTPS and SSH URLs:

```bash
./ol admin install git@github.com:open-lambda/hello-lambda-example.git
```

The function name is derived from the repository name (e.g., `hello-lambda-example`).

## Invoke Lambda

Invoke your lambda with `curl` (the result should be the same as the POST body):

```
curl -X POST localhost:5000/run/echo -d '{"hello": "world"}'
```
