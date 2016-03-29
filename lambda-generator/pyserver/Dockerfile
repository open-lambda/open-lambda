FROM alpine
RUN apk add --update python py-pip
RUN pip install rethinkdb
COPY server.py /
COPY lambda_func.py /
CMD python /server.py
