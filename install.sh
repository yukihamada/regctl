#!/bin/sh
# regctl installer — works on macOS, Linux, and WSL
# Usage: curl -fsSL https://regctl.com/install.sh | sh
set -e

REPO="yukihamada/regctl"
INSTALL_DIR="/usr/local/bin"
BINARY="regctl"

main() {
    echo ""
    echo "  Installing regctl — Domain Management CLI"
    echo "  =========================================="
    echo ""

    detect_platform
    detect_arch
    get_latest_version
    download_binary
    make_executable
    verify_install
    run_init
}

detect_platform() {
    OS="$(uname -s)"
    case "$OS" in
        Darwin)  PLATFORM="darwin" ;;
        Linux)   PLATFORM="linux" ;;
        MINGW*|MSYS*|CYGWIN*)
            PLATFORM="windows"
            BINARY="regctl.exe"
            ;;
        *)
            echo "  Unsupported OS: $OS"
            echo "  Supported: macOS, Linux, Windows (WSL/Git Bash)"
            exit 1
            ;;
    esac
    echo "  OS:       $OS ($PLATFORM)"
}

detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)
            echo "  Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    echo "  Arch:     $ARCH"
}

get_latest_version() {
    # Try to get latest release from GitHub
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"v\?\([^"]*\)".*/\1/' || echo "")
    fi

    if [ -z "$VERSION" ]; then
        VERSION="latest"
    fi
    echo "  Version:  $VERSION"
}

download_binary() {
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}-${PLATFORM}-${ARCH}"

    if [ "$PLATFORM" = "windows" ]; then
        DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
    fi

    echo ""
    echo "  Downloading from: $DOWNLOAD_URL"

    # Check for download tool
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl -fsSL -o"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget -q -O"
    else
        echo ""
        echo "  Error: curl or wget is required."
        echo ""
        echo "  Install curl:"
        echo "    macOS:  (pre-installed)"
        echo "    Ubuntu: sudo apt install curl"
        echo "    CentOS: sudo yum install curl"
        exit 1
    fi

    TMP_FILE=$(mktemp)
    if ! $DOWNLOADER "$TMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
        echo ""
        echo "  Download failed. Trying to build from source..."
        build_from_source
        return
    fi
}

build_from_source() {
    if ! command -v go >/dev/null 2>&1; then
        echo ""
        echo "  No pre-built binary available and Go is not installed."
        echo ""
        echo "  Option 1: Install Go first"
        echo "    https://go.dev/dl/"
        echo ""
        echo "  Option 2: Build manually"
        echo "    git clone https://github.com/${REPO}.git"
        echo "    cd regctl && make build"
        echo "    sudo mv regctl ${INSTALL_DIR}/"
        exit 1
    fi

    echo "  Building from source with Go..."
    go install "github.com/${REPO}/cmd/regctl@latest"
    echo "  Installed via go install."
    echo ""

    # Verify
    if command -v regctl >/dev/null 2>&1; then
        echo "  regctl $(regctl version 2>/dev/null || echo 'installed')"
        echo ""
        run_init
    fi
    exit 0
}

make_executable() {
    chmod +x "$TMP_FILE"

    # Install to INSTALL_DIR
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
    else
        echo "  Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY}"
    fi
}

verify_install() {
    echo ""
    if command -v regctl >/dev/null 2>&1; then
        echo "  Installed successfully!"
        echo "  Location: $(which regctl)"
        echo "  Version:  $(regctl version 2>/dev/null || echo 'ok')"
    else
        echo "  Binary installed to ${INSTALL_DIR}/${BINARY}"
        echo "  Make sure ${INSTALL_DIR} is in your PATH."
        echo ""
        echo "  Add to PATH (add to ~/.zshrc or ~/.bashrc):"
        echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

run_init() {
    echo ""
    echo "  ──────────────────────────────────────"
    echo ""
    echo "  Next step: Run the setup wizard"
    echo ""
    echo "    regctl init"
    echo ""
}

main
