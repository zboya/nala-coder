#!/bin/bash

# Build and test script for React + Go embedded application
set -e

echo "🚀 Starting build and test process..."

# Check prerequisites
echo "📋 Checking prerequisites..."
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js first."
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo "❌ npm is not installed. Please install npm first."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

echo "✅ Prerequisites check passed"

# Clean previous builds
echo "🧹 Cleaning previous builds..."
make clean

# Install web dependencies
echo "📦 Installing web dependencies..."
make install-web

# Build React application
echo "⚛️ Building React application..."
make build-web

# Check if React build was successful
if [ ! -d "web/dist" ]; then
    echo "❌ React build failed - dist directory not found"
    exit 1
fi

if [ ! -f "web/dist/index.html" ]; then
    echo "❌ React build failed - index.html not found"
    exit 1
fi

echo "✅ React build successful"

# Build Go binary with embedded assets
echo "🔨 Building Go binary with embedded assets..."
make build-embedded

# Check if Go build was successful
if [ ! -f "bin/nala-coder-server" ]; then
    echo "❌ Go build failed - binary not found"
    exit 1
fi

echo "✅ Go build successful"

# Check if embedded assets exist
if [ ! -d "pkg/embedded/web/dist" ]; then
    echo "❌ Embedded assets not found"
    exit 1
fi

echo "✅ Embedded assets verified"

# Get binary size
BINARY_SIZE=$(du -h bin/nala-coder-server | cut -f1)
echo "📊 Binary size: $BINARY_SIZE"

echo "🎉 Build and test completed successfully!"
echo ""
echo "To start the server:"
echo "  ./bin/nala-coder-server"
echo ""
echo "Or use:"
echo "  make server"