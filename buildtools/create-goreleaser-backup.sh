#!/usr/bin/env bash

# Create GoReleaser binary backup
# This script creates a backup of built binaries in GoReleaser naming convention
# Usage: create-goreleaser-backup.sh [VERSION]

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

# Main function
main() {
    local version=$(get_version "${1:-}")
    log_info "Creating GoReleaser binary backup for version: $version"
    
    # Create backup directory for binaries
    mkdir -p .goreleaser-binaries
    
    # Copy binaries with GoReleaser naming convention to backup location
    log_info "Copying binaries to .goreleaser-binaries/ directory..."
    
    # Linux amd64
    if [[ -f "bin/version-linux-amd64" ]]; then
        cp bin/version-linux-amd64 ".goreleaser-binaries/version_${version}_linux_amd64"
        log_info "Copied version-linux-amd64"
    else
        log_warning "bin/version-linux-amd64 not found, skipping"
    fi
    
    # Linux arm64
    if [[ -f "bin/version-linux-arm64" ]]; then
        cp bin/version-linux-arm64 ".goreleaser-binaries/version_${version}_linux_arm64"
        log_info "Copied version-linux-arm64"
    else
        log_warning "bin/version-linux-arm64 not found, skipping"
    fi
    
    # Darwin amd64
    if [[ -f "bin/version-darwin-amd64" ]]; then
        cp bin/version-darwin-amd64 ".goreleaser-binaries/version_${version}_darwin_amd64"
        log_info "Copied version-darwin-amd64"
    else
        log_warning "bin/version-darwin-amd64 not found, skipping"
    fi
    
    # Darwin arm64
    if [[ -f "bin/version-darwin-arm64" ]]; then
        cp bin/version-darwin-arm64 ".goreleaser-binaries/version_${version}_darwin_arm64"
        log_info "Copied version-darwin-arm64"
    else
        log_warning "bin/version-darwin-arm64 not found, skipping"
    fi
    
    # Windows amd64
    if [[ -f "bin/version-windows-amd64.exe" ]]; then
        cp bin/version-windows-amd64.exe ".goreleaser-binaries/version_${version}_windows_amd64.exe"
        log_info "Copied version-windows-amd64.exe"
    else
        log_warning "bin/version-windows-amd64.exe not found, skipping"
    fi
    
    log_success "GoReleaser binary backup created successfully"
    log_info "Backup location: .goreleaser-binaries/"
}

# Run main function with all arguments
main "$@"