export VARS_DIR ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

export REPO_ROOT ?= $(abspath $(VARS_DIR)/../..)
export DOCKER_DIR := docker

# Go
GOPATH ?= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOBIN ?= $(shell go env GOBIN)

# If GOBIN is not set, use GOPATH/bin
ifeq ($(GOBIN),)
	GOBIN := $(GOPATH)/bin
endif

UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
    HOST_ARCH=amd64
else ifeq ($(UNAME_M),aarch64)
    HOST_ARCH=arm64
else
    HOST_ARCH=$(UNAME_M)
endif

# Build options
export BINARY_NAME=mongo-tool
export BUILD_DIR?=$(REPO_ROOT)/build
# LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)" # Old LDFLAGS
export LDFLAGS=-ldflags "\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.appVersion=$(shell git describe --tags --always)\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.commit=$(shell git rev-parse HEAD)\
	-X github.com/drewjocham/mongo-migration-tool/internal/cli.date=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')\
"
export MAIN_PACKAGE?=./cmd

# Docker options
export DOCKER_IMAGE=mongo-tool
export DOCKER_TAG?=latest
export DOCKERFILE_LOCAL?=$(REPO_ROOT)/$(DOCKER_DIR)/Dockerfile.local
export DOCKERFILE_MCP?=$(REPO_ROOT)/$(DOCKER_DIR)/Dockerfile.mcp
export COMPOSE_FILE_INTEGRATION?=$(REPO_ROOT)/$(DOCKER_DIR)/integration-compose.yml
export COMPOSE_FILE?= $(REPO_ROOT)/$(DOCKER_DIR)/compose.yml
export COMPOSE_CMD = docker compose -f $(COMPOSE_FILE)
export DOCKER_COMPOSE ?= docker compose
export PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

export VERSION := $(shell git describe --tags --always)
export COMMIT  := $(shell git rev-parse HEAD)
export DATE    := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
export PKG     := github.com/drewjocham/mongo-migration-tool/internal/cli

export LDFLAGS := -ldflags "-s -w \
    -X $(PKG).appVersion=$(VERSION) \
    -X $(PKG).commit=$(COMMIT) \
    -X $(PKG).date=$(DATE)"

# Tooling & Versions
export GO_COMPAT_VERSION := 1.25
export GOLANGCI_VERSION := v2.6.1
export GOLANGCI_LOCAL_VERSION ?= v2.6.1
export GOLANGCI_BIN := $(GOBIN)/golangci-lint

export MOCKERY_VERSION ?= v2.53.5
export MOCKERY_BIN := $(GOBIN)/mockery

GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color
