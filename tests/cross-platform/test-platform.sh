#!/bin/bash

# Cross-platform test script for Linux distributions using buildfab
set -e

echo "=== Cross-Platform Buildfab Detection Test ==="
echo "Testing on: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo ""

# Test platform detection using buildfab
echo "Testing platform detection with buildfab..."

# Run buildfab check-detection action
echo "Running buildfab check-detection action..."
if ! ./buildfab -c unified-platform-validation.yml check-detection; then
    echo "   ❌ FAIL: buildfab check-detection failed"
    exit 1
fi
echo "   ✅ PASS"

echo ""
echo "=== All Buildfab Platform Detection Tests Passed! ==="
