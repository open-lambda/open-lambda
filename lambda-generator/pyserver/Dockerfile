FROM alpine
RUN apk add --update python
COPY server.py /
COPY lambda_func.py /
CMD python /server.py
