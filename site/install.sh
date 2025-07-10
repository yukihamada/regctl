#!/bin/bash
# regctl installer script

set -e

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Construct download URL
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/yukihamada/regctl/releases/latest/download/regctl-${OS}-${ARCH}"
else
    DOWNLOAD_URL="https://github.com/yukihamada/regctl/releases/download/${VERSION}/regctl-${OS}-${ARCH}"
fi

echo "Installing regctl..."
echo "OS: $OS"
echo "Architecture: $ARCH"
echo "Download URL: $DOWNLOAD_URL"

# Download binary
TMP_FILE=$(mktemp)
echo "Downloading..."
if command -v curl >/dev/null; then
    curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
elif command -v wget >/dev/null; then
    wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
else
    echo "Error: curl or wget is required"
    exit 1
fi

# Make executable
chmod +x "$TMP_FILE"

# Move to install directory
echo "Installing to $INSTALL_DIR/regctl..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/regctl"
else
    sudo mv "$TMP_FILE" "$INSTALL_DIR/regctl"
fi

# Verify installation
if command -v regctl >/dev/null; then
    echo "✅ regctl installed successfully!"
    regctl version
else
    echo "⚠️  regctl installed but not in PATH"
    echo "Add $INSTALL_DIR to your PATH or run: $INSTALL_DIR/regctl"
fi

# Setup completion (optional)
if [ -n "$SHELL" ]; then
    case "$SHELL" in
        */bash)
            echo ""
            echo "To enable bash completion, add this to your ~/.bashrc:"
            echo "  source <(regctl completion bash)"
            ;;
        */zsh)
            echo ""
            echo "To enable zsh completion, add this to your ~/.zshrc:"
            echo "  source <(regctl completion zsh)"
            ;;
    esac
fi

echo ""
echo "Get started with: regctl login"