#!/usr/bin/env bash

# Script to create all simple installers for all platforms and architectures
# This should be run after GoReleaser has created the distribution archives

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

# Get version from git tag or environment
get_version() {
    local version="${1:-}"
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

# Create installers for all platforms
create_all_installers() {
    local version="$1"
    local target_dir="${2:-installers}"
    
    log_info "Creating simple installers for version: $version"
    
    # Use the simple installer script
    ./buildtools/create-simple-installers.sh "$version" "$target_dir"
    
    log_success "All installers created successfully!"
}

# Main function
main() {
    local version="${1:-}"
    local target_dir="${2:-installers}"
    version=$(get_version "$version")
    
    log_info "Creating simple installers for version: $version"
    
    # Check if create-simple-installers.sh exists
    if [[ ! -f "buildtools/create-simple-installers.sh" ]]; then
        log_error "create-simple-installers.sh not found"
        exit 1
    fi
    
    create_all_installers "$version" "$target_dir"
    
    log_success "All installers created successfully!"
    log_info "Installers can be distributed via GitHub releases"
}

# Run main function with all arguments
main "$@"