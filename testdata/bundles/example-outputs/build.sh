#!/bin/bash -eu

readonly PROGDIR="$(cd "$(dirname "${0}")" && pwd)"
readonly IMAGE="carolynvs/example-outputs"
readonly VERSION="v1.0.0"

function root() {
  printf "Building the bundle"
  docker build -f "${PROGDIR}/root/Dockerfile" -t "${IMAGE}:${VERSION}" "${PROGDIR}/root"
  docker push "${IMAGE}:${VERSION}"
}

function nonroot() {
  printf "Building the nonroot flavor of the bundle"
  docker build -f "${PROGDIR}/nonroot/Dockerfile" -t "${IMAGE}:${VERSION}-nonroot" "${PROGDIR}/nonroot"
  docker push "${IMAGE}:${VERSION}-nonroot"
}

root
nonroot
