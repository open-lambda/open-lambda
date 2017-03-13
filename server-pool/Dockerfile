FROM ubuntu:trusty

RUN apt-get -y update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python python-pip build-essential
RUN pip install --upgrade pip
RUN pip install virtualenv
RUN pip install rethinkdb
RUN pip install tornado

COPY ns.so /
COPY server.py /
COPY initservers.py /

CMD ["python", "/initservers.py"]
