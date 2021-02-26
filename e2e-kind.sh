#!/usr/bin/env bash

# Adapted from https://github.com/helm/chart-testing/blob/b8572749f073372c64323618ca13255677376e0d/e2e-kind.sh

set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME=cnab-go-testing
readonly CLUSTER_NAME

K8S_VERSION=v1.19.1
readonly K8S_VERSION

KIND_KUBECONFIG="$PWD/kind-kubeconfig.yaml"
readonly KIND_KUBECONFIG

GO_TEST_COMMAND="go test -tags=integration -v -coverprofile=coverage.txt -covermode atomic ./..."
readonly GO_TEST_COMMAND

GO_TEST_LOG="go_test.log"
readonly GO_TEST_LOG

create_kind_cluster() {
    kind create cluster \
      --name "$CLUSTER_NAME" \
      --image "kindest/node:$K8S_VERSION" \
      --kubeconfig "$KIND_KUBECONFIG" \
      --wait 300s

    kubectl cluster-info --kubeconfig $KIND_KUBECONFIG
    echo

    kubectl get nodes --kubeconfig $KIND_KUBECONFIG
    echo

    echo 'Cluster ready!'
    echo
}

test_e2e() {
    echo "Running $GO_TEST_COMMAND"

    KUBECONFIG="$KIND_KUBECONFIG" $GO_TEST_COMMAND >"$GO_TEST_LOG" 2>&1
    echo
}

print_versions() {
    echo "kind version: $(kind version)"
    echo "kubectl version: $(kubectl version)"
}

cleanup() {
    cat "$GO_TEST_LOG"
    echo

    cat "$GO_TEST_LOG" | go-junit-report > report.xml
    gocov convert coverage.txt > coverage.json
    gocov-xml < coverage.json > coverage.xml

    kind delete cluster --name "$CLUSTER_NAME"
    echo 'Done!'
}

main() {
    trap cleanup EXIT

    print_versions
    create_kind_cluster
    test_e2e
}

main
