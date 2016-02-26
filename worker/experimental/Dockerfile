FROM golang:1.6-onbuild

# lay down rootfs
RUN mkdir -p /root/lambda/rootfs
COPY start-timer.tar /root/lambda/start-timer.tar
RUN tar -C /root/lambda/rootfs -xf start-timer.tar

