# use LTS
FROM ubuntu:trusty

# DOCKER (and other scripts)
RUN apt-get update -y
RUN apt-get install -y docker.io python netcat wget curl
VOLUME ["/docker_vol"]

# RETHINKDB
RUN echo "deb http://download.rethinkdb.com/apt trusty main" > /etc/apt/sources.list.d/rethinkdb.list
RUN wget -qO- https://download.rethinkdb.com/apt/pubkey.gpg | apt-key add -
RUN apt-get update
RUN apt-get install -y rethinkdb
VOLUME ["/data"]

# OL WORKER
COPY bin /open-lambda/bin

# INIT
COPY startup.py /open-lambda/startup.py
CMD ["python", "/open-lambda/startup.py"]
