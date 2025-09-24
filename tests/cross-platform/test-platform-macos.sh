#!/bin/bash

# Cross-platform test script for macOS using buildfab
set -e

echo "=== Cross-Platform Buildfab Detection Test (macOS) ==="
echo "Testing macOS platform detection with buildfab"
echo ""

# Test platform detection using buildfab
echo "Testing platform detection with buildfab..."

# Run buildfab check-detection action
echo "Running buildfab check-detection action..."
if ! ./buildfab -c macos_configuration.yml check-detection; then
    echo "   ❌ FAIL: buildfab check-detection failed on macOS"
    exit 1
fi
echo "   ✅ PASS"

echo ""
echo "=== All macOS Buildfab Platform Detection Tests Passed! ==="
