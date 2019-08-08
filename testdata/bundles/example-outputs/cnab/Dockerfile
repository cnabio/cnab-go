FROM alpine:latest

RUN apk update
RUN apk add -u bash


COPY Dockerfile /cnab/Dockerfile
COPY app /cnab/app
RUN mkdir /cnab/app/outputs

CMD ["/cnab/app/run"]
