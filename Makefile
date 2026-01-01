# Makefile for mongo-migration-tool

# Go parameters
GOCMD=go
GOMOD=$(GOCMD) mod

# Directories
BUILD_DIR=./build
CMD_DIR=./cmd

# Binary name
BINARY_NAME=mongo-migration
CMD_PATH=./cmd

# Version
VERSION ?= $(shell git describe --tags --always --dirty)

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Include other makefiles
include makefiles/*.mk

# Help target to display help messages
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Colors for echo
GREEN=\033[0;32m
NC=\033[0m
