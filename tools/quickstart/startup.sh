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

