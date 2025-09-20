#!/usr/bin/env bash
set -euo pipefail

# Script to create simple installer scripts for each platform/arch
# Usage: ./create-simple-installers.sh <version> [target_dir]

VERSION="${1:-}"
TARGET_DIR="${2:-installers}"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version> [target_dir]"
    echo "Example: $0 0.8.1"
    exit 1
fi

# Clean version (remove v prefix if present)
CLEAN_VERSION=$(echo "$VERSION" | sed 's/^v//')

# Create target directory
mkdir -p "$TARGET_DIR"

# Platforms and architectures to build
# Note: Windows uses Scoop for installation, so we don't create shell script installers for Windows
PLATFORMS=("linux" "macos")
ARCHS=("amd64" "arm64")

echo "[INFO] Creating simple installers for version $CLEAN_VERSION"

for platform in "${PLATFORMS[@]}"; do
    for arch in "${ARCHS[@]}"; do
        echo "[INFO] Creating installer for $platform-$arch"
        
        # Determine binary name and extension
        BINARY_NAME="pre-push"
        INSTALLER_NAME="pre-push-${platform}-${arch}-install.sh"
        
        # Create installer script
        INSTALLER_PATH="${TARGET_DIR}/${INSTALLER_NAME}"
        
        # Replace template variables
        sed -e "s/{{VERSION}}/$CLEAN_VERSION/g" \
            -e "s/{{PLATFORM}}/$platform/g" \
            -e "s/{{ARCH}}/$arch/g" \
            -e "s/{{BINARY_NAME}}/$BINARY_NAME/g" \
            "packaging/linux/installer-template.sh" > "$INSTALLER_PATH"
        
        # Make executable
        chmod +x "$INSTALLER_PATH"
        
        echo "[SUCCESS] Created: $INSTALLER_PATH"
    done
done

echo "[SUCCESS] All installers created in $TARGET_DIR/"
echo ""
echo "Installers can be used with:"
echo "  wget -O - <url> | sh [install_dir]"
echo "  wget -O - <url> | INSTALL_DIR=install_dir sh"
echo "  or"
echo "  ./installer.sh [install_dir]"