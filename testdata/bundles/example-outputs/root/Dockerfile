# syntax=docker/dockerfile:1.3

FROM debian:bullseye-slim
RUN mkdir -p /cnab/app/outputs
COPY run /cnab/app/run

WORKDIR /cnab/app
ENTRYPOINT /cnab/app/run
