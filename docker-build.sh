#!/bin/bash
# MenuWorks 3.X Docker Build Script
# Builds cross-platform binaries using Docker (no local Go required)

set -e

TARGET="${1:-all}"
IMAGE_NAME="menuworks-builder"

# Read version from VERSION file
if [ -f "VERSION" ]; then
    VERSION=$(cat VERSION | tr -d '\n' | xargs)
else
    echo "Error: VERSION file not found"
    exit 1
fi

echo "MenuWorks 3.X Docker Build System"
echo "Version: $VERSION"
echo "Target: $TARGET"
echo ""

# Build the Docker image
echo "Building Docker image..."
docker build -f Dockerfile.build -t "$IMAGE_NAME" .

# Ensure dist directory exists
mkdir -p dist

# Build matrix
declare -A TARGETS=(
    ["menuworks-windows.exe"]="windows/amd64"
    ["menuworks-linux"]="linux/amd64"
    ["menuworks-macos"]="darwin/amd64"
    ["menuworks-macos-arm64"]="darwin/arm64"
)

# Filter by target if specified
if [ "$TARGET" != "all" ]; then
    declare -A FILTERED
    for output in "${!TARGETS[@]}"; do
        if [[ "$output" == *"$TARGET"* ]]; then
            FILTERED["$output"]="${TARGETS[$output]}"
        fi
    done

    if [ ${#FILTERED[@]} -eq 0 ]; then
        echo "Error: No targets matched: $TARGET"
        echo "Available targets: all, windows, linux, macos"
        exit 1
    fi

    # Reassign
    unset TARGETS
    declare -A TARGETS
    for key in "${!FILTERED[@]}"; do
        TARGETS["$key"]="${FILTERED[$key]}"
    done
fi

# Build each target inside Docker
SUCCESS_COUNT=0
TOTAL_COUNT=${#TARGETS[@]}

for output in "${!TARGETS[@]}"; do
    osarch="${TARGETS[$output]}"
    os="${osarch%/*}"
    arch="${osarch#*/}"

    echo "Building $output ($os/$arch)..."

    docker run --rm \
        -e GOOS="$os" \
        -e GOARCH="$arch" \
        -v "$(pwd)/dist:/out" \
        "$IMAGE_NAME" \
        go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "/out/$output" ./cmd/menuworks

    if [ $? -eq 0 ]; then
        SIZE=$(du -h "dist/$output" | cut -f1)
        echo "  OK $output ($SIZE)"
        ((SUCCESS_COUNT++))
    else
        echo "  FAIL $output"
    fi
done

echo ""
echo "Build complete: $SUCCESS_COUNT/$TOTAL_COUNT succeeded"
echo "Output: dist/"
