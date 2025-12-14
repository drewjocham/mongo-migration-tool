VARS_MK_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(abspath $(dir $(VARS_MK_PATH))/..)
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

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
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)"

# Docker options
DOCKER_IMAGE=mongo-migration
DOCKER_TAG?=latest

# Tooling & Versions
GO_COMPAT_VERSION := 1.25
GOLANGCI_VERSION := v2.6.1
GOLANGCI_LOCAL_VERSION ?= v2.6.1
GOLANGCI_BIN := $(GOBIN)/golangci-lint

MOCKERY_VERSION ?= v2.53.5
MOCKERY_BIN := $(GOBIN)/mockery

GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color
