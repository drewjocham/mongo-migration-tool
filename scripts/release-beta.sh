#!/bin/bash

set -e

info() {
  echo -e "\033[34m[INFO]\033[0m $1"
}

success() {
  echo -e "\033[32m[SUCCESS]\033[0m $1"
}

error() {
  echo -e "\033[31m[ERROR]\033[0m $1" >&2
  exit 1
}

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# Navigate to the project root
cd "$SCRIPT_DIR/.."

# 1. Fetch the latest tags from the remote repository
info "Fetching latest tags from remote..."
git fetch --tags

# 2. Find the latest beta tag
# It looks for tags like vX.Y.Z-beta.N and sorts them to find the latest.
LATEST_BETA_TAG=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$' | head -n 1)

BASE_VERSION="v0.1.0" # Default base version if no beta tags are found
NEW_TAG=""

if [ -z "$LATEST_BETA_TAG" ]; then
  info "No existing beta tags found. Creating the first one."
  NEW_TAG="${BASE_VERSION}-beta.1"
else
  info "Latest beta tag found: $LATEST_BETA_TAG"

  # 3. Increment the beta version
  # Separate the base version from the beta number
  BASE_PART=$(echo "$LATEST_BETA_TAG" | sed -E 's/^(v[0-9]+\.[0-9]+\.[0-9]+-beta\.).*/\1/')
  BETA_NUMBER=$(echo "$LATEST_BETA_TAG" | sed -E 's/.*-beta\.([0-9]+)$/\1/')

  # Increment the beta number
  NEW_BETA_NUMBER=$((BETA_NUMBER + 1))

  # Construct the new tag
  NEW_TAG="${BASE_PART}${NEW_BETA_NUMBER}"
fi

info "New beta tag will be: $NEW_TAG"

# 4. Create and push the new tag
info "Creating and pushing new tag..."
git tag "$NEW_TAG"
git push origin "$NEW_TAG"
success "Successfully pushed tag $NEW_TAG to remote."

# 5. Run GoReleaser
info "Starting GoReleaser..."
goreleaser release --clean

success "Release process completed for $NEW_TAG."
