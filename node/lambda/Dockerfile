FROM ubuntu:trusty

RUN apt-get -y update
RUN apt-get -y install wget apt-transport-https

# RETHINKDB
RUN echo "deb https://download.rethinkdb.com/apt trusty main" > /etc/apt/sources.list.d/rethinkdb.list
RUN wget -qO- https://download.rethinkdb.com/apt/pubkey.gpg | apt-key add -
RUN apt-get update
RUN apt-get install -y rethinkdb

RUN apt-get -y install python python-pip build-essential

RUN pip install --upgrade pip
RUN pip install virtualenv
RUN pip install Flask
RUN pip install rethinkdb

# SERVER SCRIPT
COPY server.py /
COPY config.json /

CMD ["python", "/server.py"]
