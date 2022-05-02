# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

.PHONY: all test coverage
all: test coverage build

build: get
	$(GOBUILD) .

get:
	$(GOGET) -t -v ./...

fmt:
	$(GOFMT) ./...

test: get fmt
	$(GOTEST) -count=1 ./...

coverage: get test
	$(GOTEST) -coverprofile=coverage.txt -covermode=atomic .

