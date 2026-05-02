#!/bin/bash
# Generate Go code from protobuf definitions

set -e

echo "Generating Go code from proto files..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "ERROR: protoc not found. Please install Protocol Buffers compiler."
    echo "Download from: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "ERROR: protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to install protoc-gen-go"
        exit 1
    fi
fi

# Generate from mihon backup.proto
protoc --go_out=.. --go_opt=module=github.com/galpt/mk-bkconv mihon/backup.proto
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to generate Go code from mihon/backup.proto"
    exit 1
fi

echo ""
echo "✓ Successfully generated Go protobuf code"
echo "Generated files:"
echo "  - ../proto/mihon/backup.pb.go"
echo ""
