#!/bin/bash

# Build for Raspberry Pi (ARM64)
echo "Building for Raspberry Pi (ARM64)..."
GOOS=linux GOARCH=arm64 GOARM=8 CGO_ENABLED=0 go build -ldflags="-s -w" -o findvibefiber ./cmd/api

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary size: $(du -h findvibefiber | cut -f1)"
    echo "You can now copy findvibefiber to your Raspberry Pi"
else
    echo "Build failed!"
    exit 1
fi 