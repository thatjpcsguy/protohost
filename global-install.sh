#!/bin/bash
set -e

# Protohost Deploy - Global Installation Script
# Installs protohost-deploy system-wide for use across all projects

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
INSTALL_DIR="${HOME}/.local/share/protohost-deploy"
BIN_DIR="${HOME}/.local/bin"

echo "🚀 Installing protohost-deploy globally..."
echo ""

# Create installation directory
echo "📁 Creating installation directory at ${INSTALL_DIR}..."
mkdir -p "${INSTALL_DIR}/lib"
mkdir -p "${INSTALL_DIR}/templates"
mkdir -p "${BIN_DIR}"

# Copy library scripts
echo "📦 Copying library scripts..."
cp "${SCRIPT_DIR}/lib/deploy.sh" "${INSTALL_DIR}/lib/"
cp "${SCRIPT_DIR}/lib/get_ports.py" "${INSTALL_DIR}/lib/"
cp "${SCRIPT_DIR}/lib/list_deployments.sh" "${INSTALL_DIR}/lib/"
cp "${SCRIPT_DIR}/lib/nginx_manage.sh" "${INSTALL_DIR}/lib/"
chmod +x "${INSTALL_DIR}/lib/"*.sh

# Copy templates
echo "📄 Copying templates..."
cp "${SCRIPT_DIR}/templates/Makefile.template" "${INSTALL_DIR}/templates/"
cp "${SCRIPT_DIR}/templates/Makefile.inc" "${INSTALL_DIR}/templates/"
cp "${SCRIPT_DIR}/.protohost.config.example" "${INSTALL_DIR}/"
cp "${SCRIPT_DIR}/.protohost.config.local.example" "${INSTALL_DIR}/"

# Store the installation path
echo "${INSTALL_DIR}" > "${INSTALL_DIR}/.install-path"

echo "✅ Global installation complete!"
echo ""
echo "📍 Protohost installed to: ${INSTALL_DIR}"
echo ""

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
    echo "⚠️  Warning: ${BIN_DIR} is not in your PATH"
    echo "   Add this to your ~/.bashrc or ~/.zshrc:"
    echo "   export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    echo ""
fi

echo "Next steps:"
echo "  1. Run './install.sh' in your project directory to set up protohost"
echo "  2. Or run this global install on your remote server"
echo ""
