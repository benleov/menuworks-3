#!/bin/bash
# MenuWorks 3.X Docker Test Script
# Runs Go tests using Docker (no local Go required)

set -e

IMAGE_NAME="menuworks-builder"

# Default packages to test
if [ $# -gt 0 ]; then
    PACKAGES=("$@")
else
    PACKAGES=("./config" "./menu")
fi

echo "MenuWorks 3.X Docker Test Runner"
echo "Packages: ${PACKAGES[*]}"
echo ""

# Build the Docker image
echo "Building Docker image..."
docker build -f Dockerfile.build -t "$IMAGE_NAME" .
echo ""

# Run tests
FAILED=false
for pkg in "${PACKAGES[@]}"; do
    echo "Running tests for $pkg"
    if ! docker run --rm "$IMAGE_NAME" go test "$pkg"; then
        FAILED=true
    fi
done

if [ "$FAILED" = true ]; then
    echo ""
    echo "Some tests failed."
    exit 1
fi

echo ""
echo "All tests passed."
