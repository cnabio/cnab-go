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
build: fetch-schemas
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
HAS_PACKR2 := $(shell command -v packr2)

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
ifndef HAS_PACKR2
	go get -u github.com/gobuffalo/packr/v2/packr2
endif
	@# go get to install global tools with modules modify our dependencies. Reset them back
	git checkout go.mod go.sum

.PHONY: coverage
coverage:
	./e2e-kind.sh

SCHEMA_VERSION        ?= cnab-claim-1.0.0-DRAFT+d7ffba8
SCHEMA_URL_PREFIX     := https://cdn.cnab.io/schema
SCHEMA_DEST_PREFIX    := ./utils/schemavalidation/schema
BUNDLE_SCHEMA_VERSION ?= ${SCHEMA_VERSION}
CLAIM_SCHEMA_VERSION  ?= ${SCHEMA_VERSION}

.PHONY: fetch-schemas
fetch-schemas:
	@curl -s ${SCHEMA_URL_PREFIX}/${BUNDLE_SCHEMA_VERSION}/bundle.schema.json > ${SCHEMA_DEST_PREFIX}/bundle.schema.json
	@curl -s ${SCHEMA_URL_PREFIX}/${CLAIM_SCHEMA_VERSION}/claim.schema.json > ${SCHEMA_DEST_PREFIX}/claim.schema.json
