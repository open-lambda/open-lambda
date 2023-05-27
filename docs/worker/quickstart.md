# Deploying an OpenLambda Worker

## Build

OpenLambda relies heavily on operations that require root privilege.
To simplify this, we suggest that you run all commands as the root user.
OpenLambda is only actively tested on Ubuntu 22.04 LTS.

### Build and Test
Make sure you have all basic dependencies installed:
```
apt update
apt install -y docker.io llvm-12-dev libclang-common-12-dev build-essential python3 zlib1g-dev
```

For a recent version of go, run the following:
```
wget -q -O /tmp/go.tar.gz https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
ln -s /usr/local/go/bin/go /usr/bin/go
```

Further, you need to have a recent nightly version of Rust, the wasm32 toolchain, and the `cross` tool installed. The easiest way to do this is.
```
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain=nightly-2022-07-25
source $HOME/.cargo/env
rustup target add wasm32-unknown-unknown
rustup target add --toolchain=nightly-2023-02-22 wasm32-unknown-unknown
cargo install cross
```

Finally, add your user to the docker group to enable cross-compilation of native binaries to open-lambda's environment. Do not forget to restart your shell/session afterwards!
```
sudo gpasswd -a $USER docker
```

## Test

You can build the `ol` and other resources with just `make`.
Then make sure it passes the tests:

```
make test-all
```

## Run

You can create a new OL environment with the following comment:

```
./ol worker init
```

This creates a directory named `default-ol` with various OL resources.
You can create an OL environment at another location by passing a `-path=DIRNAME` to the `init` command.

Default config settings were saved to `./default-ol/config.json`.
Modify them if you wish, then start an OL worker (if you used `-path` above, use it again with the `worker` command):

```
./ol worker up
```

In another terminal, make sure the worker is running with `./ol worker status`.

Now save the following to `./default-ol/registry/echo.py`:

```python
def f(event):
    return event
```

Now invoke your lambda (the result should be the same as the POST body):

```
curl -X POST localhost:5000/run/echo -d '{"hello": "world"}'
```

When you're done, just kill the worker with `ctrl-C`.
If you want to run the worker in detached mode (i.e., in the background), just start it again with the `-d` flag:

```
./ol worker up -d
```

You can shutdown a detached worker like this:

```
./ol worker down
```
