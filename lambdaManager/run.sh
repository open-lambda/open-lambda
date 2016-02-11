docker run \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -p 8080:8080\
        --name lambda-worker \
        -d \
        lambda-worker
