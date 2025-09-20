#!/usr/bin/env bash

# Build script for pre-push Go application using Conan + CMake

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="pre-push"
BUILD_DIR="./bin"
CMAKE_BUILD_DIR=".build"
CONAN_PROFILE="default"

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

# Play success sound
play_success_sound() {
    # Try different sound methods
    if command -v paplay &> /dev/null; then
        # Try to play a system sound
        paplay /usr/share/sounds/sound-icons/prompt.wav 2>/dev/null || \
        paplay /usr/share/sounds/alsa/Front_Left.wav 2>/dev/null || \
        paplay /usr/share/sounds/alsa/Front_Right.wav 2>/dev/null || \
        paplay /usr/share/sounds/alsa/Rear_Left.wav 2>/dev/null || \
        paplay /usr/share/sounds/alsa/Rear_Right.wav 2>/dev/null || \
        echo -e "\a"  # Fallback to bell
    elif command -v aplay &> /dev/null; then
        # Try to play a system sound
        aplay /usr/share/sounds/sound-icons/prompt.wav 2>/dev/null || \
        aplay /usr/share/sounds/alsa/Front_Left.wav 2>/dev/null || \
        aplay /usr/share/sounds/alsa/Front_Right.wav 2>/dev/null || \
        aplay /usr/share/sounds/alsa/Rear_Left.wav 2>/dev/null || \
        aplay /usr/share/sounds/alsa/Rear_Right.wav 2>/dev/null || \
        echo -e "\a"  # Fallback to bell
    elif command -v speaker-test &> /dev/null; then
        # Generate a short beep
        speaker-test -t sine -f 1000 -l 1 2>/dev/null || echo -e "\a"
    else
        # Fallback to terminal bell
        echo -e "\a"
    fi
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
VERSION=""

# 1. Try to use built version utility first
if [[ -f "scripts/version" ]]; then
    VERSION=$(scripts/version version 2>/dev/null || echo "")
    if [[ -n "$VERSION" ]]; then
        log_info "Using built version utility: $VERSION"
    fi
fi

# 2. NEW: Auto-download latest version utility
if [[ -z "$VERSION" ]]; then
    if download_version_utility; then
        VERSION=$(scripts/version version 2>/dev/null || echo "")
        if [[ -n "$VERSION" ]]; then
            log_info "Using downloaded version utility: $VERSION"
        fi
    fi
fi

# 3. Fallback to git describe if version utility not available or failed
if [[ -z "$VERSION" ]]; then
    VERSION=$(git describe --match "v[0-9]*" --abbrev=0 --tags 2>/dev/null || echo "")
    if [[ -n "$VERSION" ]]; then
        log_info "Using git describe: $VERSION"
    fi
fi

if [ -z "$VERSION" ]; then
    log_error "Failed to get version from built utility, download, and git. Make sure you're in a git repository with version tags."
    exit 1
fi

# Check if Conan is installed
check_conan() {
    if ! command -v conan &> /dev/null; then
        log_error "Conan is not installed. Please install Conan 2.x"
        log_info "Install with: pip install conan"
        exit 1
    fi

    CONAN_VERSION=$(conan --version | awk '{print $2}')
    log_success "Conan version $CONAN_VERSION found"
}

# Check if CMake is installed (fallback)
check_cmake() {
    if ! command -v cmake &> /dev/null; then
        log_warning "CMake not found in PATH, will use Conan's CMake"
    else
        CMAKE_VERSION=$(cmake --version | head -n1 | awk '{print $3}')
        log_success "CMake version $CMAKE_VERSION found"
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

# Install dependencies with Conan
install_deps() {
    log_info "Installing dependencies with Conan..."
    
    # Check for golang package first
    if ! check_golang_package; then
        log_error "Failed to ensure golang package is available"
        exit 1
    fi
    
    # Create conan profile if it doesn't exist
    #if ! conan profile show $CONAN_PROFILE &> /dev/null; then
    #    log_info "Creating Conan profile: $CONAN_PROFILE"
    #    conan profile detect --force
    #fi
    
    # Install dependencies
    conan install . --build=missing --profile=$CONAN_PROFILE
    
    log_success "Dependencies installed"
}

# Configure CMake with Conan
configure_cmake() {
    local build_type="${1:-Release}"
    
    log_info "Configuring CMake with Conan (build type: $build_type)"
    
    # Create build directory
    mkdir -p "$CMAKE_BUILD_DIR"
    
    # Try to use Conan preset first (CMake 3.23+)
    if cmake --version | grep -q "3\.2[3-9]\|3\.[3-9]\|[4-9]\." && [ -f "./CMakeUserPresets.json" ]; then
        log_info "Using Conan CMake preset"
        # Convert build_type to lowercase for preset name
        preset_name="conan-$(echo $build_type | tr '[:upper:]' '[:lower:]')"
        cmake --preset $preset_name
    else
        # Fallback to manual configuration
        log_info "Using manual CMake configuration"
        cmake -B "$CMAKE_BUILD_DIR" \
              -DCMAKE_BUILD_TYPE="$build_type" \
              -DCMAKE_INSTALL_PREFIX="/usr/local" \
              -G "Unix Makefiles"
    fi
    
    log_success "CMake configured with Conan"
}

# Build using CMake + Conan
build_cmake() {
    local target="${1:-$BINARY_NAME}"
    local build_type="${2:-Release}"
    
    log_info "Building $target using CMake + Conan..."
    
    # Install dependencies if not done
    if [ ! -f "./$CMAKE_BUILD_DIR/conan_toolchain.cmake" ] && [ ! -f "./$CMAKE_BUILD_DIR/Release/generators/conan_toolchain.cmake" ]; then
        install_deps
    fi
    
    # Configure if build directory doesn't exist or doesn't have CMakeCache.txt
    if [ ! -d "./$CMAKE_BUILD_DIR" ] || [ ! -f "./$CMAKE_BUILD_DIR/CMakeCache.txt" ]; then
        configure_cmake "$build_type"
    fi
    
    # Build using preset if available
    if cmake --version | grep -q "3\.2[3-9]\|3\.[3-9]\|[4-9]\." && [ -f "./CMakeUserPresets.json" ]; then
        log_info "Building with CMake preset"
        # Convert build_type to lowercase for preset name
        preset_name="conan-$(echo $build_type | tr '[:upper:]' '[:lower:]')"
        cmake --build --preset $preset_name --target "$target"
    else
        # Fallback to manual build
        cmake --build "$CMAKE_BUILD_DIR" --target "$target"
    fi
    
    log_success "Built $target"
    play_success_sound
}

# Build with Conan directly (alternative method)
build_conan() {
    local build_type="${1:-Release}"
    
    log_info "Building with Conan directly..."
    
    # Create conan profile if it doesn't exist
    #if ! conan profile show $CONAN_PROFILE &> /dev/null; then
    #    conan profile detect --force
    #fi
    
    # Build with Conan
    conan create . --build=missing --profile=$CONAN_PROFILE -s build_type=$build_type
    
    log_success "Built with Conan"
    play_success_sound
}

# Install using Conan
install_conan() {
    local install_path="${1:-/usr/local}"
    
    log_info "Installing $BINARY_NAME using Conan..."
    
    # Create conan profile if it doesn't exist
    #if ! conan profile show $CONAN_PROFILE &> /dev/null; then
    #    conan profile detect --force
    #fi
    
    # Install with Conan
    conan install . --build=missing --profile=$CONAN_PROFILE
    
    # Copy binaries to install path
    if [ -f "$CMAKE_BUILD_DIR/$BINARY_NAME" ]; then
        sudo mkdir -p "$install_path/bin"
        sudo cp "$CMAKE_BUILD_DIR/$BINARY_NAME" "$install_path/bin/"
        log_success "Installed to $install_path/bin/"
        play_success_sound
    else
        log_error "Binary not found. Build first."
        exit 1
    fi
}

# Clean build artifacts
clean_conan() {
    log_info "Cleaning build artifacts..."
    
    # Remove build directory
    rm -rf "$CMAKE_BUILD_DIR"
    
    # Remove Conan cache (optional)
    if [ "${1:-}" = "--clean-cache" ]; then
        log_info "Cleaning Conan cache..."
        conan remove "*" --confirm
    fi
    
    # Remove other build artifacts
    rm -f coverage.out coverage.html
    
    log_success "Build artifacts cleaned"
}

# Show Conan profile info
show_profile() {
    log_info "Conan profile information:"
    conan profile show $CONAN_PROFILE
}

# Show available Conan packages
show_packages() {
    log_info "Available Conan packages:"
    conan search "*" --remote=conancenter
}

# Show help
show_help() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  deps                - Install dependencies with Conan"
    echo "  build [target]      - Build binary using CMake + Conan"
    echo "  build-conan         - Build using Conan directly"
    echo "  build-all           - Build for all platforms"
    echo "  install [path]      - Install binary (default: /usr/local)"
    echo "  install-current     - Install current OS binary to project bin directory"
    echo "  install-all         - Install all platform binaries to project bin directory"
    echo "  clean [--clean-cache] - Clean build artifacts"
    echo "  configure           - Configure CMake with Conan"
    echo "  profile             - Show Conan profile info"
    echo "  packages            - Show available Conan packages"
    echo "  help                - Show this help"
    echo ""
    echo "Build targets:"
    echo "  $BINARY_NAME                    - Build for current platform"
    echo "  $BINARY_NAME-all               - Build for all platforms"
    echo "  $BINARY_NAME-linux-amd64       - Build for Linux/amd64"
    echo "  $BINARY_NAME-linux-amd64-static - Build static binary for Linux/amd64"
    echo "  $BINARY_NAME-darwin-amd64      - Build for macOS/amd64"
    echo "  $BINARY_NAME-darwin-arm64      - Build for macOS/arm64"
    echo "  $BINARY_NAME-windows-amd64     - Build for Windows/amd64"
    echo ""
    echo "Examples:"
    echo "  $0 deps                         # Install Go and CMake via Conan"
    echo "  $0 build                        # Build for current platform"
    echo "  $0 build-conan                  # Build using Conan directly"
    echo "  $0 install                      # Install to /usr/local"
    echo "  $0 clean --clean-cache          # Clean everything including Conan cache"
}

# Main script logic
main() {
    local command="${1:-help}"
    
    # Check prerequisites
    check_conan
    check_cmake
    
    case "$command" in
        "deps")
            install_deps
            ;;
        "build")
            build_cmake "${2:-$BINARY_NAME}"
            ;;
        "build-conan")
            build_conan "${2:-Release}"
            ;;
        "build-all")
            build_cmake "$BINARY_NAME-all"
            ;;
        "install")
            install_conan "${2:-/usr/local}"
            ;;
        "install-current")
            log_info "Installing current OS binary..."
            cmake --build build/Release --target install-current
            log_success "Installed current OS binary"
            play_success_sound
            ;;
        "install-all")
            log_info "Installing all platform binaries..."
            cmake --build build/Release --target install-all
            log_success "Installed all platform binaries"
            play_success_sound
            ;;
        "clean")
            clean_conan "${2:-}"
            ;;
        "configure")
            configure_cmake "${2:-Release}"
            ;;
        "profile")
            show_profile
            ;;
        "packages")
            show_packages
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"