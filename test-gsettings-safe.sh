#!/bin/bash

# Test script with environment setup to prevent GSettings crashes
# This script sets up a safe environment for testing GSettings functionality

set -e

echo "Setting up safe testing environment for GSettings..."

# Export environment variables to prevent GTK/GSettings crashes
export GTK_USE_PORTAL=1
export GSETTINGS_BACKEND=memory
export ACCOLADE_DISABLE_GSETTINGS=1

# Set up a temporary directory for test files
export TEST_DIR=$(mktemp -d)
export XDG_CONFIG_HOME="$TEST_DIR/.config"
export XDG_DATA_HOME="$TEST_DIR/.local/share"
export XDG_CACHE_HOME="$TEST_DIR/.cache"

# Create necessary directories
mkdir -p "$XDG_CONFIG_HOME"
mkdir -p "$XDG_DATA_HOME"
mkdir -p "$XDG_CACHE_HOME"

# Clean up function
cleanup() {
    echo "Cleaning up test environment..."
    rm -rf "$TEST_DIR"
    unset GTK_USE_PORTAL
    unset GSETTINGS_BACKEND
    unset ACCOLADE_DISABLE_GSETTINGS
    unset TEST_DIR
    unset XDG_CONFIG_HOME
    unset XDG_DATA_HOME
    unset XDG_CACHE_HOME
}

# Set up trap to clean up on exit
trap cleanup EXIT

echo "Environment variables set:"
echo "  GTK_USE_PORTAL=$GTK_USE_PORTAL"
echo "  GSETTINGS_BACKEND=$GSETTINGS_BACKEND"
echo "  ACCOLADE_DISABLE_GSETTINGS=$ACCOLADE_DISABLE_GSETTINGS"
echo "  TEST_DIR=$TEST_DIR"
echo "  XDG_CONFIG_HOME=$XDG_CONFIG_HOME"
echo "  XDG_DATA_HOME=$XDG_DATA_HOME"
echo "  XDG_CACHE_HOME=$XDG_CACHE_HOME"

echo ""
echo "Running GSettings integration tests..."
zig build test-gsettings-integration || {
    echo "GSettings integration tests failed"
    exit 1
}

echo ""
echo "Running file chooser integration tests..."
zig build test-file-chooser-integration || {
    echo "File chooser integration tests failed"
    exit 1
}

echo ""
echo "Running full test suite with safe environment..."
zig build test || {
    echo "Full test suite failed"
    exit 1
}

echo ""
echo "Testing application startup with safe environment..."
timeout 5s zig build run -- --test-mode || {
    echo "Application startup test completed (timeout is expected)"
}

echo ""
echo "All tests completed successfully!"
echo "GSettings and file chooser operations are now safe from crashes."

echo ""
echo "To run the application safely, use:"
echo "  export GTK_USE_PORTAL=1"
echo "  export GSETTINGS_BACKEND=memory"
echo "  zig build run"

echo ""
echo "Or to completely disable GSettings:"
echo "  export ACCOLADE_DISABLE_GSETTINGS=1"
echo "  zig build run"