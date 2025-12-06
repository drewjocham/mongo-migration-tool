VARS_DIR ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Repository root
REPO_ROOT ?= $(abspath $(VARS_DIR)/../..)

 # Execute in one shell
.ONESHELL:

# Build options
BINARY_NAME=mongo-essential
DOCKER_IMAGE=mongo-migration-tool
DOCKER_TAG?=latest

GOOS=linux
GOARCH=amd64

BUILD_DIR=./build
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)"

# Tooling & Versions
GO_COMPAT_VERSION := 1.25
GOLANGCI_VERSION := v2.6.1
GOLANGCI_LOCAL_VERSION ?= v2.6.1
GOLANGCI_BIN := $(HOME)/go/bin/golangci-lint

MOCKERY_VERSION ?= v2.53.5
MOCKERY_BIN := $(HOME)/go/bin/mockery


GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color
