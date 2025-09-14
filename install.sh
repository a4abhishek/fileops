#!/bin/bash

# FileOps Installation Script
# This script downloads and installs the latest version of FileOps

set -e

# Default settings
INSTALL_DIR="/usr/local/bin"
REPO="a4abhishek/fileops"
BINARY_NAME="fileops"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os
    local arch
    
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name"' | \
        sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
}

# Download and install FileOps
install_fileops() {
    local version
    local platform
    local download_url
    local temp_dir
    local binary_path
    
    print_status "Detecting platform..."
    platform=$(detect_platform)
    print_status "Detected platform: ${platform}"
    
    print_status "Getting latest version..."
    version=$(get_latest_version)
    if [ -z "$version" ]; then
        print_error "Failed to get latest version"
        exit 1
    fi
    print_status "Latest version: ${version}"
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    trap "rm -rf ${temp_dir}" EXIT
    
    # Download binary
    download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${version}_${platform}.tar.gz"
    print_status "Downloading from: ${download_url}"
    
    if ! curl -L "${download_url}" -o "${temp_dir}/${BINARY_NAME}.tar.gz"; then
        print_error "Failed to download FileOps"
        exit 1
    fi
    
    # Extract binary
    print_status "Extracting binary..."
    tar -xzf "${temp_dir}/${BINARY_NAME}.tar.gz" -C "${temp_dir}"
    
    # Install binary
    binary_path="${temp_dir}/${BINARY_NAME}"
    if [ ! -f "${binary_path}" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi
    
    print_status "Installing to ${INSTALL_DIR}..."
    if [ -w "${INSTALL_DIR}" ]; then
        cp "${binary_path}" "${INSTALL_DIR}/"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        print_status "Requesting sudo permissions for installation..."
        sudo cp "${binary_path}" "${INSTALL_DIR}/"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    
    print_status "FileOps ${version} installed successfully!"
    print_status "Run 'fileops --help' to get started."
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        missing_deps+=("tar")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        print_error "Please install them and try again."
        exit 1
    fi
}

# Main installation function
main() {
    print_status "FileOps Installation Script"
    print_status "============================"
    
    check_dependencies
    install_fileops
    
    print_status ""
    print_status "🎉 Installation complete!"
    print_status ""
    print_status "Next steps:"
    print_status "  1. Run 'fileops version' to verify installation"
    print_status "  2. Run 'fileops --help' to see available commands"
    print_status "  3. Check out examples at: https://github.com/${REPO}/wiki"
}

# Run main function
main "$@"
