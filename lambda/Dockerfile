FROM ubuntu:focal

RUN apt-get -y --fix-missing update
RUN apt-get -y install wget apt-transport-https curl clang
RUN apt-get -y install python3 python3-dev python3-pip python-is-python3
RUN apt-get -y install build-essential libseccomp-dev
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==6.1.0

RUN mkdir /runtimes

# Setup rust environment (prereq for native runtime)
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly

# Build Native Runtime in the Container
RUN mkdir /runtimes/native
COPY runtimes/native /tmp/native-runtime
RUN cd /tmp/native-runtime && ~/.cargo/bin/cargo build --release
RUN mv /tmp/native-runtime/target/release/open-lambda-runtime /runtimes/native/server
RUN rm -rf /tmp/native-runtime

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
