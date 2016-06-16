#!/bin/bash
APP=pychat
add-apt-repository ppa:ubuntu-lxc/lxd-stable -y
apt-get update
apt-get -y install golang
apt-get -y install docker.io
apt-get -y install docker-engine

apt-get -y install python-pip
apt-get -y install python2.7-dev
apt-get -y install curl
apt-get -y install git
pip install netifaces
pip install rethinkdb
`clone the repository when it is public`
`git clone https://github.com/tylerharter/open-lambda`
cd open-lambda
service docker start
docker daemon
make imgs/lambda-node
./util/start-local-cluster.py
./util/setup.py pychat chat.py
docker run -d -p 80:80 -v $PWD/applications/pychat/static:/usr/share/nginx/html:ro nginx
cd ../..
echo To access app, either:
echo 1. Go to localhost:80 and use the web interface
echo 2. Use curl. Sample curl:
echo curl -w "\n" localhost:port/runLambda/imageName -d '{}'
echo port is the load balancer port found in output of docker ps
echo imageName is "latest" image from docker images

