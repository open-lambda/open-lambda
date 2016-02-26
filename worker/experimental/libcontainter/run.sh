docker run \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v /sys/fs/cgroup:/sys/fs/cgroup \
        -v /prov/self/:/prov/self/ \
        -p 8080:8080 \
        --name lambda-worker \
        --privileged=true \
        -d \
        lambda-worker
