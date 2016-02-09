# OpenLambda

# Building the lambdaWorker

### Dependancies
 - Go (https://golang.org/doc/install)
 - Docker, with a running daemon (https://docs.docker.com/engine/installation/)
 - TODO - Probably need to list the runc deps (libseccomp etc...)

```
make
ls bin/
client  lambdaWorker
```

# Building nginx

```
cd nginx
./configure --without-http_rewrite_module --without-http_gzip_module
make
```

The nginx binary is here: ./objs/nginx
