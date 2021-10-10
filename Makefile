.PHONY: help test build binary integration-test clean update-deps lint
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

GOBIN=$(GOPATH)/bin
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test
GOTOOL=$(GOCMD) tool
BINARY_NAME=clinar
TESTNAME?=.
HTMLCOVERFILE?=build/coverage.html
COVERFILE?=coverage.out
SOPS_VERSION=3.4.0

ifneq ("$(wildcard VERSION)","")
VERSION=$(shell cat VERSION)
else
VERSION=$(shell git rev-list -1 HEAD)
endif

export PATH := $(CURDIR)/.bin:$(PATH)

all: test build ## run all build and test targets

build: ## build the binary locally
	$(GOBUILD) -o $(BINARY_NAME) -ldflags "-X main.version=$(VERSION)"

install: ## build locally and copy bin into $(GOROOT)/bin
	$(GOCLEAN)
	$(GOBUILD) -o $(GOBIN)/$(BINARY_NAME) -ldflags "-X main.version=$(VERSION)"

binary: ## build the binary
	$(GOCLEAN)
	@GOOS=${GOOS} GOARCH=${GOARCH} $(GOBUILD) -a -o $(BINARY_NAME) -ldflags "-X main.version=$(VERSION)"

build-all:
	$(GOCLEAN)
	rm -rf build
	@GOOS=darwin GOARCH=amd64 GOBIN=$(GOBIN) go build \
			-tags release \
			-ldflags '-X main.version=$(VERSION)' \
			-o build/${BINARY_NAME}-${GOOS}-${GOARCH}
	tar -czvf build/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz build/${BINARY_NAME}-darwin-amd64
	@GOOS=linux GOARCH=amd64 GOBIN=$(GOBIN) go build \
			-tags release \
			-ldflags '-X main.version=$(VERSION)' \
			-o build/${BINARY_NAME}-${GOOS}-${GOARCH}
	tar -czvf build/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz build/${BINARY_NAME}-linux-amd64
	@GOOS=windows GOARCH=amd64 GOBIN=$(GOBIN) go build \
			-tags release \
			-ldflags '-X main.version=$(VERSION)' \
			-o build/${BINARY_NAME}-${GOOS}-${GOARCH}.exe
	tar -czvf build/${BINARY_NAME}-${VERSION}-win-amd64.tar.gz build/${BINARY_NAME}-windows-amd64.exe

image: binary ## build a local docker image
	docker build -t ${BINARY_NAME}:local .

test: ## run all unit tests
	$(GOTEST) -v ./... -cover -coverprofile=$(COVERFILE)

htmlcoverage: ##view coveragre after test or integration-test run
	$(GOTOOL) cover -html=build/coverage.out -o $(HTMLCOVERFILE)

clean: ## remove all compiled stuff
	$(GOCLEAN)
	rm -rf build

update-deps: ## update dependencies in vendor directory
	$(GOMOD) vendor

ci-test: test

.bin:
	mkdir .bin

lint:
	golangci-lint run --print-issued-lines=false --out-format=colored-line-number --issues-exit-code=0 --timeout 10m
