VARS_DIR ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Repository root
REPO_ROOT ?= $(abspath $(VARS_DIR)/../..)

 # Execute in one shell
.ONESHELL:
