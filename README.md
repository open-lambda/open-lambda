# OpenLambda

OpenLambda is an Apache-licensed serverless computing project, written
in Go and based on Linux containers.  The primary goal of OpenLambda
is to enable exploration of new approaches to serverless computing.  Our
research agenda is described in more detail in a [HotCloud '16
paper](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf).

## Getting Started

OpenLambda relies heavily on operations that require root
privilege. To simplify this, we suggest that you run all commands as
the root user.  OpenLambda is only actively tested on Ubuntu 16.04 LTS.

### Build and Test

OL is changing rapidly.  We recommend syncing to a commit that passed our nightly tests: https://s3.us-east-2.amazonaws.com/open-lambda-public/tests.html.

Our tests run on a VM built with this init script:
https://github.com/open-lambda/testing/blob/master/dev-build/bootstrap2.sh.
Thus, you can consider that file testable documentation of the
dependencies.

You can build the `ol` and other resources with just `make`.  Then make sure it works with some simple tests:

```
make test-all
```

### Getting Started

You can create a new OL environment with the following comment:

```
./ol new
```

This creates a directory named `default` with various OL resources.
You can create an OL environment at another location by passing a
`-path=DIRNAME` to the `new` command.

Default config settings were saved to `./default/config.json`.  Modify
them if you wish, then start an OL worker (if you used `-path` above,
use it again with the `worker` command):

```
./ol worker
```

In another terminal, make sure the worker is running with `./ol status`.

Now save the following to `./default/registry/echo.py`:

```python
def f(event):
    return event
```

Now invoke your lambda (the result should be the same as the POST body):

```
curl -X POST localhost:5000/run/echo -d '{"hello": "world"}'
```

When you're done, just kill the worker with `ctrl-C`.  If you want to
run the worker in detached mode (i.e., in the background), just start
it again with the `-d` flag:

```
./ol worker -d
```

You can shutdown a detached worker like this:

```
./ol kill
```

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.
