#!/bin/bash

# Get the version from git tag or use a default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_DIR="build"
PLATFORMS=("windows/amd64" "linux/amd64" "darwin/amd64")

# Create build directory if it doesn't exist
mkdir -p $BUILD_DIR

# Clean previous builds
rm -rf $BUILD_DIR/*

for platform in "${PLATFORMS[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    OUTPUT="gitrob-$VERSION-$GOOS-$GOARCH"
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$BUILD_DIR/$OUTPUT" -ldflags "-X github.com/BitThr3at/gitrob/core.Version=$VERSION"
    if [ $? -ne 0 ]; then
        echo "An error has occurred! Aborting..."
        exit 1
    fi
done

echo "Build complete! Binaries are in the $BUILD_DIR directory:"
ls -lh $BUILD_DIR/
