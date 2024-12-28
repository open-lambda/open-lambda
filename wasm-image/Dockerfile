FROM ol-min

RUN apt-get -y install clang

# Setup rust environment (prereq for native runtime)
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly-2024-12-16

# Build Native Runtime in the Container
RUN mkdir /runtimes/native
COPY runtimes/native /tmp/native-runtime
RUN cd /tmp/native-runtime && ~/.cargo/bin/cargo build --release
RUN mv /tmp/native-runtime/target/release/open-lambda-runtime /runtimes/native/server

# Remove build files to reduce image size
RUN rm -rf /tmp/native-runtime
RUN rm -rf ~/.cargo
RUN rm -rf ~/.rustup

CMD ["/spin"]
