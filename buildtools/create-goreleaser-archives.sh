#!/usr/bin/env bash

# Create archives in GoReleaser format
# This script creates platform-specific archives in GoReleaser naming convention
# Usage: create-goreleaser-archives.sh [VERSION]

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get version from argument or detect automatically
get_version() {
    local version="${1:-}"
    
    if [[ -n "$version" ]]; then
        echo "$version"
        return
    fi
    
    # Try to get version from built utility
    if [[ -f "bin/version" ]]; then
        version=$(bin/version version 2>/dev/null || echo "")
        if [[ -n "$version" ]]; then
            echo "$version"
            return
        fi
    fi
    
    # Fallback to VERSION file or git describe
    if [[ -f "VERSION" ]]; then
        version=$(cat VERSION)
    else
        version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    fi
    
    echo "$version"
}

# Create archive for a platform
create_platform_archive() {
    local platform="$1"
    local arch="$2"
    local binary_name="$3"
    local archive_name="$4"
    local version="$5"
    
    log_info "Creating archive for ${platform}-${arch}..."
    
    # Create temporary directory
    local temp_dir=$(mktemp -d)
    
    # Copy binary to temp directory
    if [[ -f "bin/${binary_name}" ]]; then
        if [[ "$archive_name" == *.zip ]]; then
            # Windows - keep .exe extension
            cp "bin/${binary_name}" "$temp_dir/pre-push.exe"
        else
            # Unix - no extension
            cp "bin/${binary_name}" "$temp_dir/pre-push"
        fi
        log_info "Copied ${binary_name} to temp directory"
    else
        log_warning "bin/${binary_name} not found, skipping ${platform}-${arch}"
        rm -rf "$temp_dir"
        return
    fi
    
    # Copy LICENSE and README
    if [[ -f "LICENSE" ]]; then
        cp LICENSE "$temp_dir/"
    fi
    if [[ -f "README.md" ]]; then
        cp README.md "$temp_dir/"
    fi
    
    # Create archive
    local dist_path=$(realpath dist)
    if [[ "$archive_name" == *.zip ]]; then
        # Windows zip archive
        (cd "$temp_dir" && zip -r "$dist_path/pre-push_${version}_${platform}_${arch}.zip" .)
    else
        # Unix tar.gz archive
        tar -czf "dist/pre-push_${version}_${platform}_${arch}.tar.gz" -C "$temp_dir" .
    fi
    
    # Clean up
    rm -rf "$temp_dir"
    
    log_success "Created archive: dist/pre-push_${version}_${platform}_${arch}${archive_name}"
}

# Main function
main() {
    local version=$(get_version "${1:-}")
    log_info "Creating GoReleaser archives for version: $version"
    
    # Ensure dist directory exists
    mkdir -p dist/
    
    # Create temporary directories array for cleanup
    local temp_dirs=()
    
    # Linux amd64
    create_platform_archive "linux" "amd64" "pre-push-linux-amd64" ".tar.gz" "$version"
    
    # Linux arm64
    create_platform_archive "linux" "arm64" "pre-push-linux-arm64" ".tar.gz" "$version"
    
    # Darwin amd64
    create_platform_archive "darwin" "amd64" "pre-push-darwin-amd64" ".tar.gz" "$version"
    
    # Darwin arm64
    create_platform_archive "darwin" "arm64" "pre-push-darwin-arm64" ".tar.gz" "$version"
    
    # Windows amd64
    create_platform_archive "windows" "amd64" "pre-push-windows-amd64.exe" ".zip" "$version"
    
    log_success "GoReleaser archives created successfully"
    log_info "Archives location: dist/"
    
    # List created archives
    log_info "Created archives:"
    ls -la dist/pre-push_${version}_*.tar.gz dist/pre-push_${version}_*.zip 2>/dev/null || true
}

# Run main function with all arguments
main "$@"