# syntax=docker/dockerfile:1.3

FROM debian:bullseye-slim as base
RUN mkdir -p /cnab/app/outputs
COPY run /cnab/app/run

FROM gcr.io/distroless/static:debug-nonroot
COPY --from=base --chown=65532:65532 /cnab /cnab

WORKDIR /cnab/app
USER 65532:65532
ENTRYPOINT /cnab/app/run