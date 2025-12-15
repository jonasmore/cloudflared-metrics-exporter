#!/bin/bash
set -e

# Cloudflared Metrics Exporter Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/.../install.sh | bash

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="cloudflared-metrics-exporter"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"
    
    case "$OS" in
        Linux*)
            OS="linux"
            ;;
        Darwin*)
            OS="darwin"
            ;;
        *)
            echo -e "${RED}Unsupported operating system: $OS${NC}"
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}Detected platform: ${OS}-${ARCH}${NC}"
}

# Check if running as root for system-wide install
check_permissions() {
    if [ "$INSTALL_DIR" = "/usr/local/bin" ] || [ "$INSTALL_DIR" = "/usr/bin" ]; then
        if [ "$EUID" -ne 0 ]; then
            echo -e "${YELLOW}Warning: Installing to $INSTALL_DIR requires root privileges${NC}"
            echo -e "${YELLOW}Please run with sudo or set INSTALL_DIR to a user-writable location${NC}"
            exit 1
        fi
    fi
}

# Download binary
download_binary() {
    BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_FILE}.exe"
    fi
    
    DOWNLOAD_URL="https://github.com/cloudflare/cloudflared-metrics-exporter/releases/download/${VERSION}/${BINARY_FILE}"
    
    echo -e "${GREEN}Downloading from: $DOWNLOAD_URL${NC}"
    
    TMP_DIR=$(mktemp -d)
    TMP_FILE="$TMP_DIR/$BINARY_FILE"
    
    if command -v curl &> /dev/null; then
        curl -sSL -o "$TMP_FILE" "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -q -O "$TMP_FILE" "$DOWNLOAD_URL"
    else
        echo -e "${RED}Error: curl or wget is required${NC}"
        exit 1
    fi
    
    if [ ! -f "$TMP_FILE" ]; then
        echo -e "${RED}Error: Download failed${NC}"
        exit 1
    fi
    
    chmod +x "$TMP_FILE"
    echo "$TMP_FILE"
}

# Install binary
install_binary() {
    local tmp_file=$1
    local install_path="$INSTALL_DIR/$BINARY_NAME"
    
    echo -e "${GREEN}Installing to: $install_path${NC}"
    
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Move binary
    mv "$tmp_file" "$install_path"
    
    # Verify installation
    if [ -x "$install_path" ]; then
        echo -e "${GREEN}✓ Installation successful!${NC}"
        echo ""
        echo "Binary installed to: $install_path"
        echo ""
        echo "Verify installation:"
        echo "  $BINARY_NAME --version"
        echo ""
        echo "Quick start:"
        echo "  $BINARY_NAME --metrics localhost:2000 --metricsfile /tmp/metrics.jsonl"
        echo ""
        echo "For more information, see:"
        echo "  https://github.com/cloudflare/cloudflared-metrics-exporter"
    else
        echo -e "${RED}Error: Installation failed${NC}"
        exit 1
    fi
}

# Build from source (fallback)
build_from_source() {
    echo -e "${YELLOW}Building from source...${NC}"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        echo "Please install Go 1.21+ from https://golang.org/dl/"
        exit 1
    fi
    
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    echo "Cloning repository..."
    git clone https://github.com/cloudflare/cloudflared-metrics-exporter.git
    cd cloudflared-metrics-exporter
    
    echo "Building..."
    go build -o "$BINARY_NAME"
    
    echo "$TMP_DIR/cloudflared-metrics-exporter/$BINARY_NAME"
}

# Main installation flow
main() {
    echo -e "${GREEN}=== Cloudflared Metrics Exporter Installer ===${NC}"
    echo ""
    
    detect_platform
    check_permissions
    
    # Try to download binary, fallback to building from source
    if [ "$VERSION" = "latest" ] || [ "$VERSION" = "dev" ]; then
        echo -e "${YELLOW}Building from source (no release binary available)${NC}"
        BINARY_PATH=$(build_from_source)
    else
        BINARY_PATH=$(download_binary) || {
            echo -e "${YELLOW}Download failed, falling back to building from source${NC}"
            BINARY_PATH=$(build_from_source)
        }
    fi
    
    install_binary "$BINARY_PATH"
}

# Run main function
main
