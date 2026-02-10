GOCMD=go
GOMOD=$(GOCMD) mod

BUILD_DIR=./bin
CMD_DIR=./cmd

BINARY_NAME=mongo-tool

CMD_PATH=./cmd
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

all: build

include makefiles/*.mk

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

pr-check:
	$(MAKE) build
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) integration-test
	$(MAKE) lint


GREEN=\033[0;32m
NC=\033[0m
