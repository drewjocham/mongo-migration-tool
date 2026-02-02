#!/bin/bash

set -e

VERSION=$1

if [ -z "$VERSION" ]; then
  echo "Error: Version number is required."
  echo "Usage: ./scripts/release.sh <version>"
  exit 1
fi

# Check if the version tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Error: Version tag '$VERSION' already exists."
  exit 1
fi

# Check if the working directory is clean
if ! git diff-index --quiet HEAD --; then
  echo "Error: Working directory is not clean. Please commit or stash your changes."
  exit 1
fi

# Check if on the main branch
if [ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then
  echo "Error: Not on the main branch. Please switch to main before releasing."
  exit 1
fi

echo "Creating release for version: $VERSION"

# Create the git tag
git tag -a "$VERSION" -m "Release $VERSION"

# Push the tag to the remote repository
git push origin "$VERSION"

echo "Successfully pushed tag $VERSION. The release workflow has been triggered."
