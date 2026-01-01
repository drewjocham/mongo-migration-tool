#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

if [ $# -eq 0 ]; then
  set -- release --skip=publish --snapshot --clean
fi

goreleaser --config scripts/dist/config.yaml "$@"
