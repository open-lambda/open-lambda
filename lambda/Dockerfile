FROM ubuntu:bionic

RUN apt-get -y --fix-missing update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python3 python3-dev python3-pip build-essential libseccomp-dev
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==4.5.3

# for SOCK container engine
COPY sock2.py /

# Build python module in container to avoid depending on the host
COPY ol.c /
COPY setup.py /
RUN cd / && python3 setup.py build_ext --inplace && mv ol.*.so /ol.so

# for Docker container engine
COPY server.py /
COPY spin /

CMD ["/spin"]
