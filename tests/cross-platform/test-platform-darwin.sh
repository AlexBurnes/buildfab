#!/bin/bash

# Cross-platform test script for Darwin using buildfab
set -e

echo "=== Cross-Platform Buildfab Detection Test (Darwin) ==="
echo "Testing Darwin platform detection with buildfab"
echo ""

# Test platform detection using buildfab
echo "Testing platform detection with buildfab..."

# Run buildfab check-detection action
echo "Running buildfab check-detection action..."
if ! ./buildfab -c darwin_configuration.yml check-detection; then
    echo "   ❌ FAIL: buildfab check-detection failed on Darwin"
    exit 1
fi
echo "   ✅ PASS"

echo ""
echo "=== All Darwin Buildfab Platform Detection Tests Passed! ==="
