FROM ol-min

# Setup rust environment (prereq for native runtime)
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly-2022-12-16

# Build Native Runtime in the Container
RUN mkdir /runtimes/native
COPY runtimes/native /tmp/native-runtime
RUN cd /tmp/native-runtime && ~/.cargo/bin/cargo build --release
RUN mv /tmp/native-runtime/target/release/open-lambda-runtime /runtimes/native/server
RUN rm -rf /tmp/native-runtime

CMD ["/spin"]
