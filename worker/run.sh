# Give IP of docker daemon as 1st param
docker run \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -p 8081:8080\
        --name lambda-worker \
        -e OL_DOCKER_HOST=$1 \
        -d \
        lambda-worker /go/bin/app 45.55.38.246 5000
