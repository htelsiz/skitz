#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REPO="https://github.com/htelsiz/skitz.git"

echo -e "${GREEN}Installing skitz...${NC}"

# Check for Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go from https://go.dev/dl/"
    exit 1
fi

# Check for git
if ! command -v git &> /dev/null; then
    echo -e "${RED}Error: git is not installed${NC}"
    exit 1
fi

# Determine install directory
INSTALL_DIR="/usr/local/bin"
if [[ ! -w "$INSTALL_DIR" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Create temp directory and clone
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Cloning repository..."
git clone --depth 1 "$REPO" "$TEMP_DIR" 2>/dev/null

cd "$TEMP_DIR"

# Build
echo "Building..."
go build -o skitz ./cmd/skitz/

# Install
echo "Installing to $INSTALL_DIR..."
mv skitz "$INSTALL_DIR/"

# Create config directory
CONFIG_DIR="$HOME/.config/skitz"
mkdir -p "$CONFIG_DIR/resources"

# Check if install dir is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo "Add this to your shell profile:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo -e "${GREEN}Done! Run 'skitz' to start.${NC}"
