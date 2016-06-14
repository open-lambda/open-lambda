# OpenLambda

## Building the lambdaWorker

### Dependancies
 - Go >= 1.5.3 (https://golang.org/doc/install)
 - Docker, with a running daemon (https://docs.docker.com/engine/installation/)
 - TODO - Probably need to list the runc deps (libseccomp etc...)

```
make
ls bin/
client  lambdaWorker
```

## Building nginx

```
cd nginx
./configure --without-http_rewrite_module --without-http_gzip_module
make
```

The nginx binary is here: ./objs/nginx

## Running Lambda Workers

### Dependancies
 - AUFS for Docker
 - Python Packages: rethinkdb, netifaces (requires python2.7 development headers to install)

Start OpenLabmda and then choose an application to push
```
./util/start-local-cluster.py
./applications/pychat/setup.py
```
