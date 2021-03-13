FROM ubuntu:focal

RUN apt-get update
RUN apt-get -y install wget apt-transport-https curl
RUN apt-get -y install python3 python3-dev python3-pip build-essential
RUN pip3 install --upgrade pip
RUN pip3 install virtualenv requests tornado==4.5.3

RUN mkdir /runtimes

# Build Rust Runtime in the Container
RUN mkdir /runtimes/rust
COPY runtimes/rust /tmp/rust-runtime
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly
RUN cd /tmp/rust-runtime && ~/.cargo/bin/cargo build --release
RUN mv /tmp/rust-runtime/target/release/open-lambda-runtime /runtimes/rust/server
RUN rm -rf /tmp/rust-runtime

# Build the python module also in the Container
RUN mkdir /runtimes/python
COPY runtimes/python /tmp/py-runtime
RUN cd /tmp/py-runtime && python3 setup.py build_ext --inplace
RUN mv /tmp/py-runtime/ol.*.so /runtimes/python/ol.so
RUN mv /tmp/py-runtime/server.py /runtimes/python/server.py
RUN rm -rf /tmp/py-runtime

# for the Docker container engine
COPY server.py /
COPY spin /

CMD ["/spin"]
