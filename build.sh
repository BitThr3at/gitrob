#!/bin/bash

# Get the version from git tag or use a default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_DIR="build"

# Create build directory if it doesn't exist
mkdir -p $BUILD_DIR

build_for_platform() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT=$3
    
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$BUILD_DIR/$OUTPUT" -ldflags "-X github.com/michenriksen/gitrob/core.Version=$VERSION"
}

# Clean previous builds
rm -rf $BUILD_DIR/*

# Build for different platforms
build_for_platform "linux" "amd64" "gitrob_linux_amd64"
build_for_platform "darwin" "amd64" "gitrob_darwin_amd64"
build_for_platform "windows" "amd64" "gitrob_windows_amd64.exe"

echo "Build complete! Binaries are in the $BUILD_DIR directory:"
ls -lh $BUILD_DIR/
