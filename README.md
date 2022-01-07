# OpenLambda

OpenLambda is an Apache-licensed serverless computing project, written
in Go and based on Linux containers.  The primary goal of OpenLambda
is to enable exploration of new approaches to serverless computing.
Our research agenda is described in more detail in a [HotCloud '16
paper](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf).

## Build and Test

OpenLambda relies heavily on operations that require root
privilege. To simplify this, we suggest that you run all commands as
the root user.  OpenLambda is only actively tested on Ubuntu 20.04 LTS
(AWS AMI `ami-0fb653ca2d3203ac1`, in particular).  On Ubuntu 20.04
LTS, you can install the following.

```
apt update
apt upgrade -y
apt update
apt remove -y unattended-upgrades

apt install -y python3-pip make gcc docker.io curl
pip3 install boto3

wget -q -O /tmp/go1.12.5.linux-amd64.tar.gz https://dl.google.com/go/go1.12.5.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go1.12.5.linux-amd64.tar.gz
ln -s /usr/local/go/bin/go /usr/bin/go
```

We recommend syncing to a commit that passes our daily tests:
https://s3.us-east-2.amazonaws.com/open-lambda-public/tests.html.

You can build the `ol` and other resources with just `make`.  Then
make sure it passes the tests:

```
make test-all
```

### Getting Started

You can create a new OL environment with the following comment:

```
./ol new
```

This creates a directory named `default-ol` with various OL resources.
You can create an OL environment at another location by passing a
`-path=DIRNAME` to the `new` command.

Default config settings were saved to `./default-ol/config.json`.  Modify
them if you wish, then start an OL worker (if you used `-path` above,
use it again with the `worker` command):

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
