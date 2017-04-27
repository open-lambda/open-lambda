FROM ubuntu:trusty

RUN apt-get -y update
RUN apt-get -y install wget apt-transport-https
RUN apt-get -y install python python-pip build-essential
RUN apt-get -y install python-dev
RUN pip install --upgrade pip
RUN pip install rethinkdb
RUN pip install tornado
RUN pip install posix_ipc
RUN apt-get -y update
RUN apt-get -y install linux-tools-4.4.0-72-generic
RUN apt-get -y install curl

# Protobuf install
RUN apt-get -y install git
RUN git clone https://github.com/google/protobuf.git
RUN apt-get -y install autoconf automake libtool curl make g++ unzip
RUN cd protobuf; make clean; ./autogen.sh; ./configure --prefix=/usr; make; make install; ldconfig
RUN cd protobuf/python; python setup.py build; python setup.py test; python setup.py install

COPY rand_* /


COPY server.py /
COPY init /

CMD ["python", "/server.py"]
