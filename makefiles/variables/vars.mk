VARS_DIR ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

REPO_ROOT ?= $(abspath $(VARS_DIR)/../..)

# Go environment
GOPATH ?= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOBIN ?= $(shell go env GOBIN)

# If GOBIN is not set, use GOPATH/bin
ifeq ($(GOBIN),)
	GOBIN := $(GOPATH)/bin
endif

# Build options
BINARY_NAME=mongo-migration
BUILD_DIR?=$(REPO_ROOT)/build
# LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)" # Old LDFLAGS
LDFLAGS=-ldflags "\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.appVersion=$(shell git describe --tags --always)\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.commit=$(shell git rev-parse HEAD)\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.date=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')\
"
MAIN_PACKAGE?=./cmd

# Docker options
DOCKER_IMAGE=mongo-migration
DOCKER_TAG?=latest
DOCKERFILE_LOCAL?=$(REPO_ROOT)/Dockerfile.local
DOCKERFILE_MCP?=$(REPO_ROOT)/Dockerfile.mcp
COMPOSE_FILE_INTEGRATION?=$(REPO_ROOT)/integration-compose.yml

# Tooling & Versions
GO_COMPAT_VERSION := 1.25
GOLANGCI_VERSION := v2.6.1
GOLANGCI_LOCAL_VERSION ?= v2.6.1
GOLANGCI_BIN := $(GOBIN)/golangci-lint

MOCKERY_VERSION ?= v2.53.5
MOCKERY_BIN := $(GOBIN)/mockery
export MOCKERY_VERSION
export MOCKERY_BIN

GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

export VARS_DIR
export REPO_ROOT
export BUILD_DIR
