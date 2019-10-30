GOBUILDTAGS     := osusergo

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
	go build -tags '$(GOBUILDTAGS)' ./...

.PHONY: test
test:
	go test  ./...

.PHONY: integration-test
integration-test:
	go test -tags integration ./...

.PHONY: lint
lint:
	golangci-lint run --config ./golangci.yml

HAS_DEP          := $(shell $(CHECK) dep)
HAS_GOLANGCI     := $(shell $(CHECK) golangci-lint)
HAS_GOIMPORTS    := $(shell $(CHECK) goimports)
GOLANGCI_VERSION := v1.16.0

HAS_GOCOV_XML := $(shell command -v gocov-xml;)
HAS_GOCOV := $(shell command -v gocov;)
HAS_GO_JUNIT_REPORT := $(shell command -v go-junit-report;)


.PHONY: bootstrap
bootstrap:

ifndef HAS_DEP
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
endif
ifndef HAS_GOLANGCI
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_VERSION)
endif
ifndef HAS_GOIMPORTS
	go get -u golang.org/x/tools/cmd/goimports
endif
	dep ensure -vendor-only -v

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
	go test -v -coverprofile=coverage.txt -covermode count ./... 2>&1 | go-junit-report > report.xml
	gocov convert coverage.txt > coverage.json
	gocov-xml < coverage.json > coverage.xml

.PHONY: goimports
goimports:
	find . -name "*.go" | fgrep -v vendor/ | xargs goimports -w -local github.com/$(ORG)/$(PROJECT)
