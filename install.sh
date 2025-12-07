#!/bin/bash
set -e

# Protohost Installation Script
# Can install from GitHub releases or build from source

VERSION="latest"
INSTALL_DIR="${HOME}/.local/bin"
GITHUB_REPO="thatjpcsguy/protohost"  # Update with your actual repo

echo "🚀 Installing protohost..."
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "Detected: $OS/$ARCH"
echo ""

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Check if we're in the source directory
if [ -f "go.mod" ] && grep -q "protohost" go.mod; then
    echo "📦 Building from source..."
    go build -o "${INSTALL_DIR}/protohost" cmd/protohost/main.go
    chmod +x "${INSTALL_DIR}/protohost"
else
    # Download from releases (if available)
    BINARY_NAME="protohost-${OS}-${ARCH}"

    # Try to download from GitHub releases
    # Note: Update this URL with your actual release URL
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${BINARY_NAME}.tar.gz"

    echo "📥 Downloading protohost..."
    echo "   URL: $DOWNLOAD_URL"

    # Try to download
    if command -v curl >/dev/null 2>&1; then
        if curl -fsSL "$DOWNLOAD_URL" -o "/tmp/${BINARY_NAME}.tar.gz"; then
            echo "✓ Downloaded successfully"
            tar -xzf "/tmp/${BINARY_NAME}.tar.gz" -C "/tmp"
            mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/protohost"
            chmod +x "${INSTALL_DIR}/protohost"
            rm "/tmp/${BINARY_NAME}.tar.gz"
        else
            echo "❌ Failed to download from releases"
            echo "   Please build from source or download manually"
            exit 1
        fi
    else
        echo "❌ curl not found. Please install curl or build from source"
        exit 1
    fi
fi

# Create protohost directory
mkdir -p "${HOME}/.protohost"

echo ""
echo "✅ Protohost installed to: ${INSTALL_DIR}/protohost"
echo ""

# Verify installation
if "${INSTALL_DIR}/protohost" --version >/dev/null 2>&1; then
    VERSION_OUTPUT=$("${INSTALL_DIR}/protohost" --version)
    echo "✓ Installation verified: $VERSION_OUTPUT"
else
    echo "⚠️  Installation may have issues"
fi

echo ""

# Check if in PATH
if [[ ":$PATH:" == *":${INSTALL_DIR}:"* ]]; then
    echo "✓ ${INSTALL_DIR} is already in your PATH"
else
    echo "⚠️  ${INSTALL_DIR} is not in your PATH"
    echo ""
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo "   export PATH=\"\${HOME}/.local/bin:\${PATH}\""
fi

echo ""
echo "Next steps:"
echo "  1. Ensure ${INSTALL_DIR} is in your PATH"
echo "  2. Run 'protohost init' in your project directory"
echo "  3. Deploy with 'protohost deploy'"
echo ""
