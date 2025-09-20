cmake_minimum_required(VERSION 3.16)

# Get version with corrected priority order
set(VERSION_UTILITY "${CMAKE_SOURCE_DIR}/scripts/version")
set(GIT_VERSION "")
set(PROJECT_VERSION "")

# 1. Try to use built version utility first
if(EXISTS ${VERSION_UTILITY})
    execute_process(
        COMMAND ${VERSION_UTILITY} version
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        OUTPUT_VARIABLE VERSION_UTILITY_OUTPUT
        OUTPUT_STRIP_TRAILING_WHITESPACE
        RESULT_VARIABLE VERSION_UTILITY_RESULT
    )
    
    if(VERSION_UTILITY_RESULT EQUAL 0 AND VERSION_UTILITY_OUTPUT)
        set(GIT_VERSION ${VERSION_UTILITY_OUTPUT})
        message(STATUS "Using built version utility: ${GIT_VERSION}")
    else()
        message(WARNING "Built version utility failed, trying auto-download")
    endif()
endif()

# 2. NEW: Auto-download latest version utility
if(NOT GIT_VERSION)
    message(STATUS "No version utility found, downloading latest from GitHub...")
    
    # Detect platform and architecture
    if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        set(DOWNLOAD_PLATFORM "linux")
    elseif(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
        set(DOWNLOAD_PLATFORM "macos")
    else()
        message(WARNING "Auto-download not supported on ${CMAKE_SYSTEM_NAME}, falling back to git describe")
    endif()
    
    if(CMAKE_SYSTEM_PROCESSOR STREQUAL "x86_64" OR CMAKE_SYSTEM_PROCESSOR STREQUAL "AMD64")
        set(DOWNLOAD_ARCH "amd64")
    elseif(CMAKE_SYSTEM_PROCESSOR STREQUAL "aarch64" OR CMAKE_SYSTEM_PROCESSOR STREQUAL "ARM64")
        set(DOWNLOAD_ARCH "arm64")
    else()
        message(WARNING "Auto-download not supported for ${CMAKE_SYSTEM_PROCESSOR}, falling back to git describe")
    endif()
    
    if(DOWNLOAD_PLATFORM AND DOWNLOAD_ARCH)
        message(STATUS "Detected platform: ${DOWNLOAD_PLATFORM}-${DOWNLOAD_ARCH}")
        
        # Download URL using latest release (no version number in URL)
        set(DOWNLOAD_URL "https://github.com/AlexBurnes/version-go/releases/latest/download/version-${DOWNLOAD_PLATFORM}-${DOWNLOAD_ARCH}.tar.gz")
        
        # Create scripts directory
        file(MAKE_DIRECTORY ${CMAKE_SOURCE_DIR}/scripts)
        
        # Download and install using pipe approach
        execute_process(
            COMMAND bash -c "wget -q -O - ${DOWNLOAD_URL} | INSTALL_DIR=${CMAKE_SOURCE_DIR}/scripts sh"
            WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
            RESULT_VARIABLE DOWNLOAD_RESULT
        )
        
        if(DOWNLOAD_RESULT EQUAL 0)
            if(EXISTS ${VERSION_UTILITY})
                # Try to use downloaded version utility
                execute_process(
                    COMMAND ${VERSION_UTILITY} version
                    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
                    OUTPUT_VARIABLE VERSION_UTILITY_OUTPUT
                    OUTPUT_STRIP_TRAILING_WHITESPACE
                    RESULT_VARIABLE VERSION_UTILITY_RESULT
                )
                
                if(VERSION_UTILITY_RESULT EQUAL 0 AND VERSION_UTILITY_OUTPUT)
                    set(GIT_VERSION ${VERSION_UTILITY_OUTPUT})
                    message(STATUS "Using downloaded version utility: ${GIT_VERSION}")
                endif()
            endif()
        endif()
    endif()
endif()

# 3. Fallback to git describe if version utility not available or failed
if(NOT GIT_VERSION)
    execute_process(
        COMMAND git describe --match "v[0-9]*" --abbrev=0 --tags
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        OUTPUT_VARIABLE GIT_VERSION
        OUTPUT_STRIP_TRAILING_WHITESPACE
        RESULT_VARIABLE GIT_VERSION_RESULT
    )
    
    if(GIT_VERSION_RESULT EQUAL 0 AND GIT_VERSION)
        message(STATUS "Using git describe: ${GIT_VERSION}")
    else()
        message(FATAL_ERROR "Failed to get version from built utility, download, and git describe. Make sure you're in a git repository.")
    endif()
endif()

# Extract base version (major.minor.patch) for CMake VERSION field
string(REGEX MATCH "^v?([0-9]+\\.[0-9]+\\.[0-9]+)" BASE_VERSION "${GIT_VERSION}")
if(BASE_VERSION)
    set(PROJECT_VERSION ${CMAKE_MATCH_1})
    set(FULL_GIT_VERSION ${GIT_VERSION})
    message(STATUS "Parsed version: ${GIT_VERSION} -> ${PROJECT_VERSION}")
else()
    message(FATAL_ERROR "Could not parse version: ${GIT_VERSION}. Expected format: vX.Y.Z or X.Y.Z")
endif()

# Project configuration
project(buildfab
    VERSION ${PROJECT_VERSION}
    DESCRIPTION "Go-based automation runner with DAG executor"
    LANGUAGES NONE
)

# Create scripts directory and build version utility for self-building
file(MAKE_DIRECTORY ${CMAKE_SOURCE_DIR}/scripts)

# Map CMake system to Go platform
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    set(GO_OS "linux")
elseif(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    set(GO_OS "darwin")
elseif(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    set(GO_OS "windows")
else()
    set(GO_OS "${CMAKE_SYSTEM_NAME}")
endif()

if(CMAKE_SYSTEM_PROCESSOR STREQUAL "x86_64")
    set(GO_ARCH "amd64")
elseif(CMAKE_SYSTEM_PROCESSOR STREQUAL "AMD64")
    set(GO_ARCH "amd64")
elseif(CMAKE_SYSTEM_PROCESSOR STREQUAL "aarch64")
    set(GO_ARCH "arm64")
elseif(CMAKE_SYSTEM_PROCESSOR STREQUAL "ARM64")
    set(GO_ARCH "arm64")
else()
    set(GO_ARCH "${CMAKE_SYSTEM_PROCESSOR}")
endif()

# Include Conan generated files (if they exist)
if(EXISTS "${CMAKE_BINARY_DIR}/conan_toolchain.cmake")
    include(${CMAKE_BINARY_DIR}/conan_toolchain.cmake)
elseif(EXISTS "${CMAKE_BINARY_DIR}/Release/generators/conan_toolchain.cmake")
    include(${CMAKE_BINARY_DIR}/Release/generators/conan_toolchain.cmake)
endif()

# Set Go version requirement
set(GO_VERSION_REQUIRED "1.22")

# Find Go (will use Conan's Go if available)
find_program(GO_EXECUTABLE go REQUIRED)
if(NOT GO_EXECUTABLE)
    message(FATAL_ERROR "Go compiler not found. Please install Go ${GO_VERSION_REQUIRED} or later.")
endif()

# Version utility is already available in scripts/version

# Check Go version
execute_process(
    COMMAND ${GO_EXECUTABLE} version
    OUTPUT_VARIABLE GO_VERSION_OUTPUT
    OUTPUT_STRIP_TRAILING_WHITESPACE
)

# Extract version number
string(REGEX MATCH "go([0-9]+\\.[0-9]+)" GO_VERSION "${GO_VERSION_OUTPUT}")
set(GO_VERSION_NUMBER ${CMAKE_MATCH_1})

# Check if version is sufficient
if(GO_VERSION_NUMBER VERSION_LESS GO_VERSION_REQUIRED)
    message(FATAL_ERROR "Go version ${GO_VERSION_NUMBER} is too old. Required: ${GO_VERSION_REQUIRED} or later.")
endif()

message(STATUS "Found Go version: ${GO_VERSION_NUMBER}")

# Build configuration
set(BINARY_NAME "buildfab")
set(BUILD_DIR "${CMAKE_SOURCE_DIR}/bin")
set(CONFIG_DIR "${CMAKE_SOURCE_DIR}/etc")

# Create build directory
file(MAKE_DIRECTORY ${BUILD_DIR})

# Set build flags
set(GO_BUILD_FLAGS
    -trimpath
    -ldflags "-X main.appVersion=${FULL_GIT_VERSION} -s -w"
)

# Set test flags
set(GO_TEST_FLAGS
    -v
    -race
    -coverprofile=coverage.out
)

# Platform-specific build configurations
set(PLATFORMS
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Default build target (current platform) - static build for Linux
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    add_custom_target(${BINARY_NAME}
        COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 GOOS=linux GOARCH=amd64
        ${GO_EXECUTABLE} build -a -ldflags "-extldflags \"-static\"" ${GO_BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64-static .
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/pre-push
        COMMENT "Building static ${BINARY_NAME} for current platform (linux/amd64)"
        VERBATIM
    )
else()
    add_custom_target(${BINARY_NAME}
        COMMAND ${GO_EXECUTABLE} build ${GO_BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/pre-push
        COMMENT "Building ${BINARY_NAME} for current platform"
        VERBATIM
    )
endif()

# Cross-compilation targets
foreach(PLATFORM ${PLATFORMS})
    string(REPLACE "/" ";" PLATFORM_LIST ${PLATFORM})
    list(GET PLATFORM_LIST 0 GOOS)
    list(GET PLATFORM_LIST 1 GOARCH)
    
    set(BINARY_SUFFIX "")
    if(GOOS STREQUAL "windows")
        set(BINARY_SUFFIX ".exe")
    endif()
    
    set(BINARY_NAME_PLATFORM "${BINARY_NAME}-${GOOS}-${GOARCH}${BINARY_SUFFIX}")
    
    # Build static binaries for darwin and linux, regular for windows
    if(GOOS STREQUAL "darwin" OR GOOS STREQUAL "linux")
        add_custom_target(${BINARY_NAME}-${GOOS}-${GOARCH}
            COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH}
            ${GO_EXECUTABLE} build -a -ldflags "-extldflags \"-static\"" ${GO_BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME_PLATFORM} .
            WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/pre-push
            COMMENT "Building static ${BINARY_NAME} for ${GOOS}/${GOARCH}"
            VERBATIM
        )
    else()
        add_custom_target(${BINARY_NAME}-${GOOS}-${GOARCH}
            COMMAND ${CMAKE_COMMAND} -E env GOOS=${GOOS} GOARCH=${GOARCH}
            ${GO_EXECUTABLE} build ${GO_BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME_PLATFORM} .
            WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/pre-push
            COMMENT "Building ${BINARY_NAME} for ${GOOS}/${GOARCH}"
            VERBATIM
        )
    endif()
endforeach()

# Static Linux build target for maximum compatibility
add_custom_target(${BINARY_NAME}-linux-amd64-static
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 GOOS=linux GOARCH=amd64
    ${GO_EXECUTABLE} build -a -ldflags "-extldflags \"-static\"" ${GO_BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/version
    COMMENT "Building static ${BINARY_NAME} for linux/amd64 (maximum compatibility)"
    DEPENDS bootstrap-version
    VERBATIM
)

# Build all platforms target
add_custom_target(${BINARY_NAME}-all
    DEPENDS ${BINARY_NAME}-linux-amd64
    DEPENDS ${BINARY_NAME}-linux-arm64
    DEPENDS ${BINARY_NAME}-darwin-amd64
    DEPENDS ${BINARY_NAME}-darwin-arm64
    DEPENDS ${BINARY_NAME}-windows-amd64
    COMMENT "Building ${BINARY_NAME} for all platforms"
)

# Test target
add_custom_target(test
    COMMAND ${GO_EXECUTABLE} test ${GO_TEST_FLAGS} .
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/version
    COMMENT "Running tests"
    VERBATIM
)

# Test with coverage target
add_custom_target(test-coverage
    COMMAND ${GO_EXECUTABLE} test ${GO_TEST_FLAGS} .
    COMMAND ${GO_EXECUTABLE} tool cover -html=coverage.out -o coverage.html
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/version
    COMMENT "Running tests with coverage"
    VERBATIM
)

# Format target
add_custom_target(format
    COMMAND ${GO_EXECUTABLE} fmt ./...
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Formatting code"
    VERBATIM
)

# Lint target
find_program(GOLANGCI_LINT_EXECUTABLE golangci-lint)
if(GOLANGCI_LINT_EXECUTABLE)
    add_custom_target(lint
        COMMAND ${GOLANGCI_LINT_EXECUTABLE} run
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}/cmd/pre-push
        COMMENT "Linting code"
        VERBATIM
    )
else()
    add_custom_target(lint
        COMMAND ${CMAKE_COMMAND} -E echo "golangci-lint not found, skipping linting"
        COMMENT "Linting code (skipped - golangci-lint not found)"
        VERBATIM
    )
endif()

# Dependencies target
add_custom_target(deps
    COMMAND ${GO_EXECUTABLE} mod download
    COMMAND ${GO_EXECUTABLE} mod tidy
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Downloading dependencies"
    VERBATIM
)

# Clean target
add_custom_target(clean-build
    COMMAND ${CMAKE_COMMAND} -E remove_directory ${BUILD_DIR}
    COMMAND ${CMAKE_COMMAND} -E remove -f coverage.out coverage.html
    COMMENT "Cleaning build artifacts"
    VERBATIM
)

# Install current target - install current OS binary to project bin directory
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    add_custom_target(install-current
        COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
        COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-linux-amd64-static ${BUILD_DIR}/
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
        DEPENDS ${BINARY_NAME}
        COMMENT "Installing current OS binary (static Linux) to project bin directory"
        VERBATIM
    )
elseif(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    if(CMAKE_SYSTEM_PROCESSOR STREQUAL "arm64")
        add_custom_target(install-current
            COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
            COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
            COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${BUILD_DIR}/
            COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
            DEPENDS ${BINARY_NAME}
            COMMENT "Installing current OS binary (macOS ARM64) to project bin directory"
            VERBATIM
        )
    else()
        add_custom_target(install-current
            COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
            COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
            COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${BUILD_DIR}/
            COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
            DEPENDS ${BINARY_NAME}
            COMMENT "Installing current OS binary (macOS Intel) to project bin directory"
            VERBATIM
        )
    endif()
elseif(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    add_custom_target(install-current
        COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
        COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BUILD_DIR}/
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
        DEPENDS ${BINARY_NAME}
        COMMENT "Installing current OS binary (Windows) to project bin directory"
        VERBATIM
    )
else()
    add_custom_target(install-current
        COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
        COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME} ${BUILD_DIR}/
        COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
        DEPENDS ${BINARY_NAME}
        COMMENT "Installing current OS binary to project bin directory"
        VERBATIM
    )
endif()

# Install all target - install all platform binaries to project bin directory
add_custom_target(install-all
    COMMAND ${CMAKE_COMMAND} -E make_directory ${BUILD_DIR}
    COMMAND ${CMAKE_COMMAND} -E make_directory ${CONFIG_DIR}
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${BUILD_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-linux-amd64-static ${BUILD_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${BUILD_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${BUILD_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BUILD_DIR}/
    COMMAND ${CMAKE_COMMAND} -E copy_if_different ${CMAKE_CURRENT_SOURCE_DIR}/config/version.yaml ${CONFIG_DIR}/
    DEPENDS ${BINARY_NAME}-all
    COMMENT "Installing all platform binaries to project bin directory"
    VERBATIM
)

# Development target
add_custom_target(dev
    DEPENDS deps ${BINARY_NAME}
    COMMENT "Development build"
)

# Run target
add_custom_target(run
    COMMAND ${BUILD_DIR}/${BINARY_NAME} --help
    DEPENDS ${BINARY_NAME}
    COMMENT "Running ${BINARY_NAME}"
    VERBATIM
)

# Package target
set(CPACK_PACKAGE_NAME "version")
set(CPACK_PACKAGE_VERSION ${PROJECT_VERSION})
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY ${PROJECT_DESCRIPTION})
set(CPACK_PACKAGE_VENDOR "Svyazcom")
set(CPACK_PACKAGE_CONTACT "support@svyazcom.ru")

# Create package
include(CPack)

# Custom package target
add_custom_target(create-package
    COMMAND ${CMAKE_COMMAND} --build ${CMAKE_BINARY_DIR} --target package
    COMMENT "Creating package"
    VERBATIM
)

# Show configuration
message(STATUS "Project: ${PROJECT_NAME} ${PROJECT_VERSION}")
message(STATUS "Go executable: ${GO_EXECUTABLE}")
message(STATUS "Go version: ${GO_VERSION_NUMBER}")
message(STATUS "Build directory: ${BUILD_DIR}")
message(STATUS "Config directory: ${CONFIG_DIR}")

# Print available targets
message(STATUS "")
message(STATUS "Available targets:")
message(STATUS "  ${BINARY_NAME}           - Build for current platform (static for Linux)")
message(STATUS "  ${BINARY_NAME}-all       - Build for all platforms")
message(STATUS "  ${BINARY_NAME}-linux-amd64        - Build for Linux/amd64")
message(STATUS "  ${BINARY_NAME}-linux-amd64-static - Build static binary for Linux/amd64 (max compatibility)")
message(STATUS "  ${BINARY_NAME}-darwin-amd64       - Build for macOS/amd64")
message(STATUS "  ${BINARY_NAME}-darwin-arm64       - Build for macOS/arm64")
message(STATUS "  ${BINARY_NAME}-windows-amd64      - Build for Windows/amd64")
message(STATUS "  test                    - Run tests")
message(STATUS "  test-coverage           - Run tests with coverage")
message(STATUS "  format                  - Format code")
message(STATUS "  lint                    - Lint code")
message(STATUS "  deps                    - Download dependencies")
message(STATUS "  clean-build             - Clean build artifacts")
message(STATUS "  install-current         - Install current OS binary to project bin directory")
message(STATUS "  install-all             - Install all platform binaries to project bin directory")
message(STATUS "  dev                     - Development build")
message(STATUS "  run                     - Run the application")
message(STATUS "  create-package          - Create package")
message(STATUS "")