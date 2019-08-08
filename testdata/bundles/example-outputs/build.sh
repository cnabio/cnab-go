#!/bin/bash -eu

readonly PROGDIR="$(cd "$(dirname "${0}")" && pwd)"
readonly IMAGE="pvtlmc/example-outputs"

function main() {
  docker build -f "${PROGDIR}/cnab/Dockerfile" -t "${IMAGE}:latest" "${PROGDIR}/cnab"

  local image
  image=$(docker inspect --format='{{index .RepoDigests 0}}' ${IMAGE})

  printf "\nSet DOCKER_INTEGRATION_TEST_IMAGE=%s\nto override the image used in the docker integration tests\n" "${image}"
}

main
