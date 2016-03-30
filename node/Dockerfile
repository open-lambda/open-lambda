FROM lambda-worker:latest

# DOCKER (and other scripts)
# Installing docker in jessie is fun (https://docs.docker.com/engine/installation/linux/debian/)
RUN apt-get update -y
RUN apt-get install -y apt-transport-https ca-certificates
RUN apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
RUN echo "deb https://apt.dockerproject.org/repo debian-jessie main" >> /etc/apt/sources.list.d/docker.list
RUN apt-get update -y
RUN apt-get install -y docker-engine
VOLUME ["/docker_vol"]

# rethink deps
RUN apt-get install -y python netcat wget lsb-release

# RETHINKDB
# Installing rethinkdb in jessie is also fun (https://www.rethinkdb.com/docs/install/debian/)
RUN echo "deb http://download.rethinkdb.com/apt `lsb_release -cs` main" > /etc/apt/sources.list.d/rethinkdb.list
RUN wget -qO- https://download.rethinkdb.com/apt/pubkey.gpg | apt-key add -
RUN apt-get update -y
RUN apt-get install -y rethinkdb
VOLUME ["/data"]

# OL WORKER
RUN mkdir -p /open-lambda/bin
RUN cp /go/bin/app /open-lambda/bin/worker

# INIT
COPY startup.py /open-lambda/startup.py
CMD ["python", "/open-lambda/startup.py"]
