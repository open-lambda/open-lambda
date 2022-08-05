# OpenLambda

[![CI](https://github.com/open-lambda/open-lambda/actions/workflows/ci.yml/badge.svg)](https://github.com/open-lambda/open-lambda/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

OpenLambda is an Apache-licensed serverless computing project, written (mostly) in Go and based on Linux containers.
The primary goal of OpenLambda is to enable exploration of new approaches to serverless computing.
Our research agenda is described in more detail in a [HotCloud '16 paper](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf).

## Build and Test

OpenLambda relies heavily on operations that require root privilege.
To simplify this, we suggest that you run all commands as the root user.
OpenLambda is only actively tested on Ubuntu 22.04 LTS.

### Build and Test
Make sure you have all basic dependencies installed:
```
apt install docker.io llvm-12-dev libclang-common-12-dev build-essential python3
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
cargo install cross
```

Finally, add your user to the docker group to enable cross-compilation of native binaries to open-lambda's environment. Do not forget to restart your shell/session afterwards!
```
sudo gpasswd -a $USER docker
```

You can build the `ol` and other resources with just `make`.
Then make sure it passes the tests:

```
make test-all
```

<!-- Currently broken.
We recommend syncing to a commit that passes our [daily tests](https://s3.us-east-2.amazonaws.com/open-lambda-public/tests.html).
-->

### Getting Started

You can create a new OL environment with the following comment:

```
./ol new
```

This creates a directory named `default-ol` with various OL resources.
You can create an OL environment at another location by passing a `-path=DIRNAME` to the `new` command.

Default config settings were saved to `./default-ol/config.json`.
Modify them if you wish, then start an OL worker (if you used `-path` above, use it again with the `worker` command):

```
./ol worker
```

In another terminal, make sure the worker is running with `./ol
status`.

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
./ol worker -d
```

You can shutdown a detached worker like this:

```
./ol kill
```

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.
