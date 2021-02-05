# cnab-go

[![Build Status](https://dev.azure.com/deislabs/cnab-go/_apis/build/status/cnab-go?branchName=main)](https://dev.azure.com/deislabs/cnab-go/_build/latest?definitionId=27&branchName=main) ![Azure Pipelines coverage](https://img.shields.io/azure-devops/coverage/deislabs/cnab-go/27/main?logo=Azure%20Pipelines)

cnab-go is a library for building [CNAB](https://github.com/cnabio/cnab-spec) clients. It provides the building blocks relevant to the CNAB specification so that you may build tooling without needing to implement all aspects of the CNAB specification.

cnab-go is currently being used by [Docker App](https://github.com/docker/app), [Duffle](https://github.com/cnabio/duffle), and [Porter](https://github.com/deislabs/porter). If you'd like to see your CNAB project listed here, please submit a PR.

### Community

cnab-go is [maintained](GOVERNANCE.md) by the CNAB community. We sometimes discuss cnab-go issues during the [bi-weekly CNAB community meeting](https://hackmd.io/s/SyGcBcwQ4), but we encourage open communication via our [issue](https://github.com/cnabio/cnab-go/issues) queue and via [PRs](https://github.com/cnabio/cnab-go/pulls). If you are interested in contributing to cnab-go, please refer to our [contributing](CONTRIBUTING.md) guidelines.

### Development

#### Getting the code

Cloning this repository and change directory to it:

```bash
$ go get -d github.com/cnabio/cnab-go/...
$ cd $(go env GOPATH)/src/github.com/cnabio/cnab-go
```

#### Prerequisites

You need:

- make
- Go

#### Get dependencies

Retrieve all needed packages to start developing.
This will download the binaries for the linter, dep and go imports in the end it will
run `dep ensure` to download all the go package dependencies

```bash
$ make bootstrap
```

#### Building, testing and linting

Compile all the code:

```bash
$ make build
```

Run unit tests:

```bash
$ make test
```

Run integration tests:

Note: The integration tests use a local docker daemon to create containers. Please ensure you have one configured before running the tests.

```bash
$ make integration-test
```

This will only run the linter to ensure the code meet the standard.
_It does not format the code_

```bash
$ make lint
```
