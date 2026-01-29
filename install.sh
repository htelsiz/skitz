#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

VERSION="0.1.0"
REPO="https://github.com/htelsiz/skitz.git"

# Print colored text
print_color() {
    echo -e "${1}${2}${NC}"
}

# Progress bar
progress_bar() {
    local current=$1
    local total=$2
    local width=50
    local percent=$((current * 100 / total))
    local filled=$((width * current / total))
    local empty=$((width - filled))

    printf "\r${YELLOW}"
    printf "%${filled}s" | tr ' ' '▓'
    printf "%${empty}s" | tr ' ' '░'
    printf "${NC} ${BOLD}%3d%%${NC}" $percent
}

# ASCII art logo with crane
print_logo() {
    echo ""
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⡿⠟⠋⣁⡄⠀⢠⣄⣉⡙⠛⠿⢿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⠿⠛⣁⣤⣶⣿⠇⣤⠈⣿⣿⣿⣿⣶⣦⣄⣉⠙⠛⠿   '"${BOLD}█▀ █▄▀ █ ▀█▀ ▀█${NC}"
    print_color "$PURPLE" '⣿⣿⣯⣤⣴⣿⣿⣿⣿⣿⣤⣿⣤⣽⣿⣿⣿⣿⣿⣿⣿⣿⣷⣦   '"${BOLD}▄█ █ █ █  █  █▄${NC}"
    print_color "$PURPLE" '⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿'
    print_color "$PURPLE" '⣿⣿⣿⡟⠛⠛⠛⣿⣿⣿⣿⡟⠛⢻⡟⠛⢻⣿⣿⣿⣿⣿⣿⣿   '"${DIM}v${VERSION} Command Center${NC}"
    print_color "$PURPLE" '⣿⣿⣿⣷⣶⣶⣶⣿⣿⣿⣿⣇⣀⣸⣇⣀⣼⣿⣿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡏⠉⢹⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⡇⠀⢸⡿⣿⣿⣿⣿⠀⠀⠀⢸⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⡿⠋⣁⣴⡇⠀⢸⣷⣌⠙⢿⣿⣿⣿⣿⣿⣿'
    print_color "$PURPLE" '⣿⣿⣿⣿⣿⣿⣿⣷⣾⣿⣿⣷⣤⣼⣿⣿⣿⣶⣿⣿⣿⣿⣿⣿'
    echo -e "          ${YELLOW}▟${NC}\033[30;43m B I A \033[0m${YELLOW}▙${NC}"
    echo ""
}

# Detect shell profile
detect_shell_profile() {
    if [[ -n "$ZSH_VERSION" ]] || [[ "$SHELL" == *"zsh"* ]]; then
        echo "$HOME/.zshrc"
    elif [[ -n "$BASH_VERSION" ]] || [[ "$SHELL" == *"bash"* ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            echo "$HOME/.bash_profile"
        else
            echo "$HOME/.bashrc"
        fi
    else
        echo "$HOME/.profile"
    fi
}

# Main installation
main() {
    echo ""
    print_color "$BOLD" "Installing ${PURPLE}skitz${NC}${BOLD} version: ${VERSION}${NC}"

    # Step 1: Check dependencies
    progress_bar 1 5
    sleep 0.2

    if ! command -v go &> /dev/null; then
        echo ""
        print_color "$RED" "Error: Go is not installed"
        print_color "$DIM" "Please install Go from https://go.dev/dl/"
        exit 1
    fi

    if ! command -v git &> /dev/null; then
        echo ""
        print_color "$RED" "Error: git is not installed"
        exit 1
    fi

    # Step 2: Setup directories
    progress_bar 2 5
    sleep 0.2

    INSTALL_DIR="/usr/local/bin"
    if [[ ! -w "$INSTALL_DIR" ]]; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    fi

    CONFIG_DIR="$HOME/.config/skitz"
    DATA_DIR="$HOME/.local/share/skitz"
    mkdir -p "$CONFIG_DIR/resources"
    mkdir -p "$DATA_DIR"

    # Step 3: Clone repository
    progress_bar 3 5

    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    GIT_TERMINAL_PROMPT=0 git clone --depth 1 --quiet "$REPO" "$TEMP_DIR" 2>/dev/null

    # Step 4: Build
    progress_bar 4 5

    cd "$TEMP_DIR"
    go build -ldflags="-s -w" -o skitz . 2>/dev/null

    # Step 5: Install
    mv skitz "$INSTALL_DIR/"

    progress_bar 5 5
    echo ""

    # Add to PATH if needed
    SHELL_PROFILE=$(detect_shell_profile)
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_PROFILE"
        print_color "$GREEN" "Successfully added ${BOLD}skitz${NC}${GREEN} to \$PATH in ${SHELL_PROFILE}${NC}"
    else
        print_color "$GREEN" "Successfully installed ${BOLD}skitz${NC}${GREEN} to ${INSTALL_DIR}${NC}"
    fi

    # Print logo and instructions
    print_logo

    print_color "$DIM" "Terminal command center with MCP integration"
    echo ""
    print_color "$CYAN" "To get started:"
    echo ""
    print_color "$BOLD" "  skitz              ${DIM}# Launch dashboard${NC}"
    print_color "$BOLD" "  skitz claude       ${DIM}# Open Claude resource${NC}"
    print_color "$BOLD" "  skitz --help       ${DIM}# Show help${NC}"
    echo ""
    print_color "$DIM" "For more information visit ${CYAN}https://github.com/htelsiz/skitz${NC}"
    echo ""

    # Remind about new shell if PATH was modified
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_color "$YELLOW" "Note: Restart your shell or run: source ${SHELL_PROFILE}"
    fi
}

main "$@"
