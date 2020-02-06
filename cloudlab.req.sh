
# Install oh-my-bash
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmybash/oh-my-bash/master/tools/install.sh)"

sudo apt update

# Install dependencies
sudo apt -y install linux-modules-extra-$(uname -r)
sudo apt -y install linux-image-extra-virtual
sudo apt -y install curl
sudo apt -y install git
sudo apt -y install docker.io
sudo apt -y install cgroup-tools cgroup-bin
sudo apt -y install python2.7-dev
sudo apt -y install python-pip
sudo apt -y install python3-pip
service docker restart

# python (2+3)
pip install netifaces
pip install rethinkdb
pip install tornado
pip3 install boto3

# go 1.12.5
wget -q -O /tmp/go1.12.5.linux-amd64.tar.gz https://dl.google.com/go/go1.12.5.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go1.12.5.linux-amd64.tar.gz
ln -s /usr/local/go/bin/go /usr/bin/go

# disable auto updates
sudo apt-get -y remove unattended-upgrades
