#!/usr/bin/env bash

# Post-archive hook script for GoReleaser
# This script creates simple installers after archives are created

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

# Get version from environment or git
get_version() {
    local version="${GORELEASER_CURRENT_TAG:-}"
    if [[ -z "$version" ]]; then
        # Try to use built version utility first
        if [[ -f "scripts/version" ]]; then
            version=$(scripts/version version 2>/dev/null || echo "")
        fi
        
        # Fallback to git describe if version utility not available or failed
        if [[ -z "$version" ]]; then
            if command -v git >/dev/null 2>&1; then
                version=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
            else
                version="unknown"
            fi
        fi
    fi
    echo "$version"
}

# Main function
main() {
    local version
    version=$(get_version)
    
    log_info "Creating simple installers for version: $version"
    
    # Check if dist directory exists
    if [[ ! -d "dist" ]]; then
        log_error "dist directory not found"
        exit 1
    fi
    
    # Check if create-all-installers.sh exists
    if [[ ! -f "buildtools/create-all-installers.sh" ]]; then
        log_error "create-all-installers.sh not found"
        exit 1
    fi
    
    # Create installers
    ./buildtools/create-all-installers.sh "$version"
    
    log_success "Simple installers created successfully!"
}

# Run main function
main "$@"