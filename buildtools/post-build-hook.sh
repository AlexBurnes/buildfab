#!/bin/bash
# Post-build hook to replace GoReleaser-built binaries with our pre-built ones

set -e

echo "[INFO] Post-build hook: Replacing GoReleaser binaries with pre-built ones..."

# Check if pre-built binaries exist
if [ ! -d ".goreleaser-binaries" ]; then
    echo "[WARNING] No pre-built binaries found in .goreleaser-binaries/"
    exit 0
fi

# Copy pre-built binaries to dist/ directory
echo "[INFO] Copying pre-built binaries to dist/..."
cp .goreleaser-binaries/* dist/ 2>/dev/null || true

echo "[SUCCESS] Pre-built binaries copied to dist/"