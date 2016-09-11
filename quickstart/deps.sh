#!/bin/bash
add-apt-repository ppa:ubuntu-lxc/lxd-stable -y
apt-get update
apt-get -y install linux-image-extra-$(uname -r)
apt-get -y install linux-image-extra-virtual
apt-get -y install golang
apt-get -y install docker.io
apt-get -y install docker-engine

apt-get -y install python-pip
apt-get -y install python2.7-dev
apt-get -y install curl
apt-get -y install git
pip install netifaces
pip install rethinkdb
service docker restart
