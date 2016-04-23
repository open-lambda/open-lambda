FROM alpine
RUN apk add --update python py-pip
RUN pip install rethinkdb
RUN pip install Flask
COPY server.py /
COPY lambda_func.py /
COPY config.json /
CMD python /server.py
