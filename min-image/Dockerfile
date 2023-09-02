FROM ubuntu:22.04

RUN apt-get -y --fix-missing update
RUN apt-get -y install wget apt-transport-https curl
RUN apt-get -y install python3 python3-dev python3-pip python-is-python3
RUN apt-get -y install build-essential libseccomp-dev
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==6.1.0

RUN mkdir /runtimes

# Build the python module also in the Container
RUN mkdir /runtimes/python
COPY runtimes/python /tmp/py-runtime
RUN cd /tmp/py-runtime && python3 setup.py build_ext --inplace
RUN mv /tmp/py-runtime/ol.*.so /runtimes/python/ol.so
RUN mv /tmp/py-runtime/server.py /runtimes/python/server.py
RUN mv /tmp/py-runtime/server_legacy.py /runtimes/python/server_legacy.py
RUN rm -rf /tmp/py-runtime

# for the Docker container engine
COPY spin /

CMD ["/spin"]
