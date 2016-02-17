docker run \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -p 8080:8080\
        --name lambda-worker \
        -d \
        lambda-worker /go/bin/app 45.55.38.246 5000
