#!/bin/bash

# Check if we're on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS, setting up cross-compilation environment..."
    
    # Set up cross-compilation environment
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=arm64
    export GOARM=8
else
    # For Linux, use native compilation
    export CGO_ENABLED=1
    export GOOS=linux
    export GOARCH=arm64
    export GOARM=8
fi

echo "Building for Raspberry Pi (ARM64)..."
go build -ldflags="-s -w" -o findvibefiber ./cmd

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary size: $(du -h findvibefiber | cut -f1)"
    echo "You can now copy findvibefiber to your Raspberry Pi"
    
    # Create a deployment package
    echo "Creating deployment package..."
    mkdir -p deploy
    cp findvibefiber deploy/
    cp findvibefiber.service deploy/
    
    echo "Deployment package created in 'deploy' directory"
    echo "To deploy on Raspberry Pi:"
    echo "1. Copy the contents of the 'deploy' directory to your Raspberry Pi"
    echo "2. Run: sudo mv findvibefiber.service /etc/systemd/system/"
    echo "3. Run: sudo systemctl daemon-reload"
    echo "4. Run: sudo systemctl enable findvibefiber"
    echo "5. Run: sudo systemctl start findvibefiber"
else
    echo "Build failed!"
    exit 1
fi 