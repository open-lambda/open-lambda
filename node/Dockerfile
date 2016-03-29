FROM debian:jessie
RUN apt-get update

# DOCKER
RUN echo 'deb http://ftp.debian.org/debian jessie-backports main' >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y docker.io python
VOLUME ["/docker_vol"]

# RETHINKDB
RUN apt-key adv --keyserver pgp.mit.edu --recv-keys 1614552E5765227AEC39EFCFA7E00EF33A8F2399
RUN echo "deb http://download.rethinkdb.com/apt jessie main" > /etc/apt/sources.list.d/rethinkdb.list
ENV RETHINKDB_PACKAGE_VERSION 2.2.5~0jessie
RUN apt-get update \
    && apt-get install -y rethinkdb=$RETHINKDB_PACKAGE_VERSION \
    && rm -rf /var/lib/apt/lists/*
VOLUME ["/data"]

# OL WORKER
COPY bin /open-lambda/bin

# INIT
COPY startup.py /open-lambda/startup.py
CMD ["python", "/open-lambda/startup.py"]
