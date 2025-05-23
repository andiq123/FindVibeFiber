#!/bin/bash

echo "Building for Raspberry Pi (native build)..."
go build -ldflags="-s -w" -o findvibefiber ./cmd

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary size: $(du -h findvibefiber | cut -f1)"
    
    # Create a deployment package
    echo "Creating deployment package..."
    mkdir -p deploy
    cp findvibefiber deploy/
    cp findvibefiber.service deploy/
    
    echo "Deployment package created in 'deploy' directory"
    echo "To set up the service:"
    echo "1. Run: sudo mv findvibefiber.service /etc/systemd/system/"
    echo "2. Run: sudo systemctl daemon-reload"
    echo "3. Run: sudo systemctl enable findvibefiber"
    echo "4. Run: sudo systemctl start findvibefiber"
else
    echo "Build failed!"
    exit 1
fi 