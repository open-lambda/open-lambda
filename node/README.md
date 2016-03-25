To run a worker node, do the following:

docker run -d -p 5000:5000 registry:2
cd lambda-generator
./builder.py --name=my-lambda
cd ..
make node
docker run --privileged --net=host -v /sys/fs/cgroup:/sys/fs/cgroup lambda-node

Now, from another node, run this:

curl -X POST localhost:8080/runLambda/my-lambda -d '{}'
