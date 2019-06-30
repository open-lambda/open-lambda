FROM ubuntu:bionic

RUN apt-get -y --fix-missing update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python3 python3-dev python3-pip build-essential
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==4.5.3

# for SOCK container engine
COPY sock2.py /
COPY ol.so /

# for Docker container engine
COPY server.py /
COPY spin /

CMD ["/spin"]
