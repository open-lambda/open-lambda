FROM ubuntu
RUN apt-get update
RUN apt-get install -y docker.io python

COPY bin /open-lambda/bin
COPY startup.py /open-lambda/startup.py

VOLUME ["/docker_vol"]
CMD ["python", "/open-lambda/startup.py"]
