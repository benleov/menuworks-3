#!/bin/bash
# MenuWorks 3.0 Build Script (Unix/Linux/macOS)
# Builds cross-platform binaries using local Go installation

TARGET="${1:all}"
VERSION="${2:-1.0.0}"

# Set local Go path
LOCAL_GO="$(pwd)/bin/go/bin/go"
if [ ! -f "$LOCAL_GO" ]; then
    echo "Error: Go not found at $LOCAL_GO"
    echo "Please install Go in bin/go first."
    exit 1
fi

export PATH="$(pwd)/bin/go/bin:$PATH"

echo "MenuWorks 3.0 Build System"
echo "Go: $LOCAL_GO"
echo "Version: $VERSION"
echo ""

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
    
    TARGETS=("${FILTERED[@]}")
fi

# Build each target
SUCCESS_COUNT=0
TOTAL_COUNT=${#TARGETS[@]}

for output in "${!TARGETS[@]}"; do
    osarch="${TARGETS[$output]}"
    os="${osarch%/*}"
    arch="${osarch#*/}"
    
    echo "Building $output ($os/$arch)..."
    
    export GOOS="$os"
    export GOARCH="$arch"
    
    LD_FLAGS="-X main.version=$VERSION"
    OUTPUT_PATH="dist/$output"
    
    "$LOCAL_GO" build -ldflags "$LD_FLAGS" -o "$OUTPUT_PATH" cmd/menuworks/main.go
    
    if [ $? -eq 0 ]; then
        SIZE=$(du -h "$OUTPUT_PATH" | cut -f1)
        echo "  ✓ $output ($SIZE)"
        ((SUCCESS_COUNT++))
    else
        echo "  ✗ Failed to build $output"
    fi
done

echo ""
echo "Build complete: $SUCCESS_COUNT/$TOTAL_COUNT targets succeeded"

# Clean environment
unset GOOS
unset GOARCH

if [ $SUCCESS_COUNT -eq $TOTAL_COUNT ]; then
    echo "All builds successful!"
    exit 0
else
    exit 1
fi
