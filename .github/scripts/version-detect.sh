#!/usr/bin/env bash
# Version detection script for RC versioning

# Strict error handling
set -euo pipefail

# Determine latest stable version (handle case where no tags exist)
LAST_STABLE_VERSION=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname 2>/dev/null | grep -v '-' | head -n1 || true)

# Bootstrap case: no stable tags
if [ -z "$LAST_STABLE_VERSION" ]; then
    LAST_STABLE_VERSION="v0.0.0"
fi

# Extract version components
MAJOR=$(echo "$LAST_STABLE_VERSION" | cut -d. -f1 | sed 's/v//')
MINOR=$(echo "$LAST_STABLE_VERSION" | cut -d. -f2)
PATCH=$(echo "$LAST_STABLE_VERSION" | cut -d. -f3)

# Increment patch version
NEXT_PATCH=$((PATCH + 1))
NEXT_VERSION="v${MAJOR}.${MINOR}.${NEXT_PATCH}"

# Count existing RC tags for this version
RC_COUNT=$(git tag --list "${NEXT_VERSION}-rc.*" | wc -l | tr -d ' ')
NEXT_RC_NUMBER=$((RC_COUNT + 1))

# Construct next RC version
NEXT_RC_VERSION="${NEXT_VERSION}-rc.${NEXT_RC_NUMBER}"

# Output for GitHub Actions
echo "NEXT_RC_VERSION=${NEXT_RC_VERSION}" >> "$GITHUB_OUTPUT"
echo "NEXT_RC_NUMBER=${NEXT_RC_NUMBER}" >> "$GITHUB_OUTPUT"
echo "LAST_STABLE_VERSION=${LAST_STABLE_VERSION}" >> "$GITHUB_OUTPUT"