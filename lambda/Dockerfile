FROM ubuntu:trusty

RUN apt-get -y update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python python-pip build-essential
RUN pip install --upgrade pip
RUN pip install rethinkdb

# for getting namespace pid and forkenter
COPY init /

CMD ["./init"]
