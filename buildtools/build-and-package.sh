#!/usr/bin/env bash

# Build script using build-conan.sh + simple installers + goreleaser (skip build)
# Strategy: build-conan -> simple installers -> goreleaser (skip build, only package/publish)

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

# Load environment variables from .env file
load_environment() {
    if [[ -f ".env" ]]; then
        log_info "Loading environment variables from .env file..."
        # Source the .env file, ignoring comments and empty lines
        set -a  # automatically export all variables
        source .env
        set +a  # disable automatic export
        log_success "Environment variables loaded from .env"
    else
        log_warning "No .env file found, using system environment variables only"
    fi
}

# Setup Go environment
setup_go_environment() {
    log_info "Setting up Go environment..."
    
    # Get Go's GOPATH and add bin directory to PATH
    local go_bin_dir
    go_bin_dir=$(go env GOPATH)/bin
    
    if [[ -d "$go_bin_dir" ]]; then
        export PATH="$go_bin_dir:$PATH"
        log_info "Added Go bin directory to PATH: $go_bin_dir"
    else
        log_warning "Go bin directory not found: $go_bin_dir"
    fi
}

# Check for golang package in Conan
check_golang_package() {
    log_info "Checking for golang package in Conan..."
    
    if conan search golang --remote=all 2>/dev/null | grep -q "golang/"; then
        log_success "golang package found in Conan remote repositories"
        return 0
    else
        log_warning "golang package not found in Conan remote repositories"
        log_info "Creating golang package locally..."
        
        # Check if conanfile-golang.py exists
        if [ ! -f "conanfile-golang.py" ]; then
            log_error "conanfile-golang.py not found in project root"
            log_info "Please ensure conanfile-golang.py exists before running this script"
            return 1
        fi
        
        # Create golang package locally
        log_info "Creating golang package from local recipe..."
        if conan create conanfile-golang.py --build=missing; then
            log_success "golang package created locally and available for use"
            return 0
        else
            log_error "Failed to create golang package locally"
            return 1
        fi
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check GoReleaser
    if ! command -v goreleaser &> /dev/null; then
        log_error "GoReleaser is not installed. Install with: go install github.com/goreleaser/goreleaser@latest"
        exit 1
    fi
    
    # Check Conan
    if ! command -v conan &> /dev/null; then
        log_error "Conan is not installed. Install with: pip install conan"
        exit 1
    fi
    
    # Check Git
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed"
        exit 1
    fi
    
    # Check for golang package
    if ! check_golang_package; then
        log_error "Failed to ensure golang package is available"
        exit 1
    fi
    
    log_success "All prerequisites found"
}

# Download version utility from GitHub releases
download_version_utility() {
    log_info "No version utility found, downloading latest from GitHub..."
    
    # Detect platform and architecture
    local platform=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    # Map architecture names
    case "$arch" in
        x86_64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) log_error "Unsupported architecture: $arch"; return 1 ;;
    esac
    
    # Map platform names
    case "$platform" in
        darwin) platform="macos" ;;
        linux) platform="linux" ;;
        *) log_error "Unsupported platform: $platform"; return 1 ;;
    esac
    
    log_info "Detected platform: ${platform}-${arch}"
    
    # Download URL using latest release (no version number in URL)
    local download_url="https://github.com/AlexBurnes/version-go/releases/latest/download/version-${platform}-${arch}.tar.gz"
    
    # Create scripts directory
    mkdir -p scripts
    
    # Download and install using pipe approach
    log_info "Downloading version utility from: $download_url"
    if wget -q -O - "$download_url" | INSTALL_DIR="$(dirname "$0")/scripts" sh; then
        if [[ -f "scripts/version" ]]; then
            log_success "Successfully downloaded version utility"
            return 0
        else
            log_error "Version binary not found after installation"
            return 1
        fi
    else
        log_error "Failed to download and install version utility from GitHub"
        return 1
    fi
}

# Get version with corrected priority order
get_version() {
    local version=""
    
    # 1. Try to use built version utility first
    if [[ -f "scripts/version" ]]; then
        version=$(scripts/version version 2>/dev/null || echo "")
        if [[ -n "$version" ]]; then
            echo "$version"
            return
        fi
    fi
    
    # 2. NEW: Auto-download latest version utility
    if download_version_utility; then
        version=$(scripts/version version 2>/dev/null || echo "")
        if [[ -n "$version" ]]; then
            echo "$version"
            return
        fi
    fi
    
    # 3. Fallback to VERSION file or git describe
    if [[ -f "VERSION" ]]; then
        version=$(cat VERSION)
    else
        version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    fi
    
    # Clean version (remove v prefix for some operations)
    echo "$version"
}

# Step 1: Build binaries using build-conan.sh
build_binaries() {
    log_info "Step 1: Building binaries using build-conan.sh..."
    
    # Clean previous builds
    rm -rf dist/
    mkdir -p dist/
    
    # Build for all platforms using build-conan.sh
    ./buildtools/build-conan.sh build-all
    
    # Store binaries in a backup location for GoReleaser hooks
    log_info "Storing binaries for GoReleaser hooks..."
    ./buildtools/create-goreleaser-backup.sh
    
    log_success "Binaries stored for GoReleaser hooks"
    
    # Create archives in GoReleaser format for install scripts
    log_info "Creating archives in GoReleaser format..."
    ./buildtools/create-goreleaser-archives.sh
    
    log_success "Archives created in dist/ directory"
}

# Step 2: Create install scripts
create_install_scripts() {
    local version="${1:-}"
    log_info "Step 2: Creating install scripts..."
    
    # Clean installers directory to remove old installers
    rm -rf installers/
    
    # Create simple installers in installers/ directory
    ./buildtools/create-all-installers.sh "$version" "installers"
    
    log_success "Install scripts created"
}

# Step 3: Publish with GoReleaser (builds binaries, then replaces with pre-built ones)
publish_release() {
    local mode="${1:-snapshot}"
    
    log_info "Step 3: Publishing release with GoReleaser..."
    
    case "$mode" in
        "snapshot")
            goreleaser release --snapshot --clean
            ;;
        "release")
            goreleaser release --clean
            ;;
        "dry-run")
            goreleaser release --snapshot --skip=publish --clean
            ;;
        *)
            log_error "Unknown mode: $mode"
            exit 1
            ;;
    esac
    
    log_success "Release published successfully"
}

# Clean build artifacts
clean_build() {
    log_info "Cleaning build artifacts..."
    rm -rf dist/
    rm -rf bin/
    rm -rf installers/
    rm -rf .goreleaser-binaries/
    rm -f .goreleaser-skip-build.yml
    log_success "Build artifacts cleaned (including old installers)"
}

# Show help
show_help() {
    echo "Usage: $0 [MODE]"
    echo ""
    echo "Modes:"
    echo "  snapshot          # Build snapshot with pre-built binaries"
    echo "  release           # Build full release with pre-built binaries"
    echo "  dry-run           # Build dry run with pre-built binaries"
    echo "  clean             # Clean build artifacts"
    echo "  help              # Show this help message"
    echo ""
    echo "Strategy:"
    echo "  1. Build binaries using build-conan.sh (static for darwin/linux)"
    echo "  2. Create simple install scripts"
    echo "  3. Publish with GoReleaser (skip build, only package/publish)"
    echo ""
    echo "Examples:"
    echo "  $0 snapshot          # Build snapshot with install scripts"
    echo "  $0 release           # Build full release with install scripts"
    echo "  $0 dry-run           # Build dry run with install scripts"
    echo "  $0 clean             # Clean build artifacts"
}

# Main script logic
main() {
    local mode="${1:-snapshot}"
    
    # Change to project root directory
    cd "$(dirname "$0")/.."
    
    # Load environment and setup
    load_environment
    setup_go_environment
    check_prerequisites
    
    case "$mode" in
        "clean")
            clean_build
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        "snapshot"|"release"|"dry-run")
            log_info "Starting build process with build-conan.sh (mode: $mode)..."
            
            # Get version
            local version=$(get_version)
            log_info "Using version: $version"
            
            # Step 1: Build binaries
            build_binaries
            
            # Step 2: Create install scripts
            create_install_scripts "$version"
            
            # Step 3: Publish release (skip build)
            if [[ "$mode" != "dry-run" ]]; then
                publish_release "$mode"
            else
                log_info "Dry-run mode: skipping publish step"
            fi
            
            log_success "Build process completed successfully!"
            log_info "Install scripts should now be included in the GitHub release"
            ;;
        *)
            log_error "Unknown mode: $mode"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"