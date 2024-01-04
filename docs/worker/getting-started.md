# Deploying an OpenLambda Worker

## Dependencies

OpenLambda relies heavily on operations that require root privilege.
To simplify this, we suggest that you run all commands as the root user.

OpenLambda is only actively tested on Ubuntu 22.04 LTS.  If you try
running on a different distro, one thing you'll definitely need is
cgroups2 mounted at /sys/fs/cgroup (`mount | grep cgroup2`).
OpenLambda does not work with cgroups v1.

Make sure you have all basic dependencies installed:
```
apt update
apt install -y docker.io llvm-12-dev libclang-common-12-dev build-essential python3 zlib1g-dev
```

For a recent version of go, run the following:
```
wget -q -O /tmp/go.tar.gz https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
ln -s /usr/local/go/bin/go /usr/bin/go
```

### Optional: Full Deployment (with WebAssembly Support)

OpenLambda has support for both Python-based and WebAssemby-based
lambdas (e.g., you could create a lambda by compiling Rust to
WebAssembly).

You can skip this part if you only want Python (a "min" deployment).

If want to write Rust-based lambdas, you need to have a recent nightly
version of Rust, the wasm32 toolchain, and the `cross` tool
installed. The easiest way to do this is:

```
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain=nightly-2022-07-25
source $HOME/.cargo/env
rustup target add wasm32-unknown-unknown
rustup target add --toolchain=nightly-2023-02-22 wasm32-unknown-unknown
cargo install cross
```

Finally, add your user to the docker group to enable cross-compilation
of native binaries to OpenLambda's environment. Do not forget to
restart your shell/session afterwards!

```
sudo gpasswd -a $USER docker
```

## Build

For a full deployment with Python+WASM support, just run `make all`.

For a "min" deployment (just Python), run `make ol imgs/ol-min`.

## Test

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

## Creating a Lambda

Now save the following to `./default-ol/registry/echo.py`:

```python
def f(event):
    return event
```

## Invoke Lambda

Invoke your lambda with `curl` (the result should be the same as the POST body):

```
curl -X POST localhost:5000/run/echo -d '{"hello": "world"}'
```
