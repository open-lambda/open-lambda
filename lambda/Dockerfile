FROM ubuntu:bionic

RUN apt-get update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python3 python3-dev python3-pip build-essential
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==4.5.3

# for SOCK container engine
RUN mkdir /runtimes
COPY runtimes/python /runtimes/python

# Build python module in container to avoid depending on the host
RUN cd /runtimes/python && python3 setup.py build_ext --inplace && mv ol.*.so ol.so

RUN mkdir /runtimes/rust
COPY runtimes/rust/target/release/open-lambda-runtime /runtimes/rust/

# for Docker container engine
COPY server.py /
COPY spin /

CMD ["/spin"]
