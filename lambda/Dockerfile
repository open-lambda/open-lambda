FROM ubuntu:trusty

RUN apt-get -y update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python python-dev python-pip build-essential
RUN pip install --upgrade pip
RUN pip install rethinkdb
RUN pip install tornado

COPY server.py /
COPY ol-init /

CMD ["python", "/server.py"]
