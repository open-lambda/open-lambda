FROM ubuntu:trusty

RUN apt-get -y --fix-missing update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python python-dev python-pip build-essential
RUN pip install --upgrade pip
RUN pip install virtualenv
RUN pip install rethinkdb
RUN pip install tornado==4.5.3
RUN pip install requests

COPY spin /
COPY ol-init /
COPY ns.so /
COPY server.py /

CMD ["/spin"]
