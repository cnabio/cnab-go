ifeq ($(OS),Windows_NT)
	TARGET = $(PROJECT).exe
	SHELL  = cmd.exe
	CHECK  = where.exe
else
	TARGET = $(PROJECT)
	SHELL  ?= bash
	CHECK  ?= which
endif

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run --config ./golangci.yml

HAS_GOLANGCI := $(shell $(CHECK) golangci-lint)
GOLANGCI_VERSION := v1.21.0
HAS_KIND := $(shell $(CHECK) kind)
HAS_KUBECTL := $(shell $(CHECK) kubectl)
HAS_GOCOV_XML := $(shell command -v gocov-xml;)
HAS_GOCOV := $(shell command -v gocov;)
HAS_GO_JUNIT_REPORT := $(shell command -v go-junit-report;)


.PHONY: bootstrap
bootstrap:

ifndef HAS_GOLANGCI
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_VERSION)
endif
ifndef HAS_KIND
	go get sigs.k8s.io/kind@v0.6.0
endif
ifndef HAS_KUBECTL
	echo "Follow instructions at https://kubernetes.io/docs/tasks/tools/install-kubectl/ to install kubectl."
endif
ifndef HAS_GOCOV_XML
	go get github.com/AlekSi/gocov-xml
endif
ifndef HAS_GOCOV
	go get -u github.com/axw/gocov/gocov
endif
ifndef HAS_GO_JUNIT_REPORT
	go get github.com/jstemmer/go-junit-report
endif

.PHONY: coverage
coverage:
	./e2e-kind.sh
