#!/bin/bash

set -e

echo "=== Accolade Fyne Build Script ==="
echo "Building the Go/Fyne version of Accolade"
echo

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    echo "Please install Go 1.21 or later"
    exit 1
fi

# Show Go version
echo "Go version: $(go version)"
echo

# Check for required system libraries
echo "Checking system dependencies..."

missing_libs=()

# Check for X11 development libraries
if ! pkg-config --exists x11 2>/dev/null; then
    missing_libs+=("libx11-dev")
fi

if ! pkg-config --exists xrandr 2>/dev/null; then
    missing_libs+=("libxrandr-dev")
fi

if ! pkg-config --exists xcursor 2>/dev/null; then
    missing_libs+=("libxcursor-dev")
fi

if ! pkg-config --exists xinerama 2>/dev/null; then
    missing_libs+=("libxinerama-dev")
fi

if ! pkg-config --exists xi 2>/dev/null; then
    missing_libs+=("libxi-dev")
fi

if ! pkg-config --exists gl 2>/dev/null; then
    missing_libs+=("libgl1-mesa-dev")
fi

# Check if gcc is available
if ! command -v gcc &> /dev/null; then
    missing_libs+=("build-essential")
fi

# Check if pkg-config is available
if ! command -v pkg-config &> /dev/null; then
    missing_libs+=("pkg-config")
fi

if [ ${#missing_libs[@]} -ne 0 ]; then
    echo "Error: Missing required system dependencies:"
    printf ' - %s\n' "${missing_libs[@]}"
    echo
    echo "On Ubuntu/Debian, install them with:"
    echo "sudo apt-get update"
    echo "sudo apt-get install ${missing_libs[*]}"
    echo
    echo "On Fedora/RHEL, install them with:"
    echo "sudo dnf install libX11-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel mesa-libGL-devel gcc pkg-config"
    echo
    echo "On Arch Linux, install them with:"
    echo "sudo pacman -S libx11 libxrandr libxcursor libxinerama libxi mesa gcc pkg-config"
    exit 1
fi

echo "✓ All system dependencies found"
echo

# Set up environment for CGO
export CGO_ENABLED=1

# Set up build flags for better optimization
export CGO_CFLAGS="-O2 -g"
export CGO_LDFLAGS="-O2 -g"

# Clean any previous builds
echo "Cleaning previous builds..."
go clean -cache
go clean -modcache || true  # Don't fail if no modcache
rm -f accolade

echo "Downloading dependencies..."
go mod tidy
go mod download

echo "Building Accolade..."
go build -v -ldflags="-s -w -X main.Version=$(git describe --tags --always 2>/dev/null || echo 'dev')" -o accolade .

if [ $? -eq 0 ]; then
    echo
    echo "✓ Build successful!"
    echo "Executable: ./accolade"
    echo "Size: $(du -h accolade | cut -f1)"
    echo
    echo "To run:"
    echo "  ./accolade"
    echo
    echo "To install system-wide:"
    echo "  sudo cp accolade /usr/local/bin/"
    echo "  sudo chmod +x /usr/local/bin/accolade"
else
    echo
    echo "✗ Build failed!"
    exit 1
fi

# Optional: Run a quick test
if [ "$1" = "--test" ]; then
    echo "Running quick test..."
    timeout 5s ./accolade --help 2>/dev/null || echo "Note: GUI application, no command-line help available"
fi

echo
echo "=== Build Complete ==="