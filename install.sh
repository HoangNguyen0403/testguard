#!/usr/bin/env bash
set -euo pipefail

REPO="HoangNguyen0403/testguard"
PROJECT="testguard"

echo "🛡️ Installing TestGuard..."

# 1. Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)
    echo "❌ Unsupported operating system: ${OS}. Supported: darwin, linux"
    exit 1
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: ${ARCH}. Supported: amd64, arm64"
    exit 1
    ;;
esac

# 3. Determine install destination
INSTALL_DIR="/usr/local/bin"
USE_SUDO=false

if [ ! -w "${INSTALL_DIR}" ]; then
  if command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    USE_SUDO=true
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
  fi
fi

# 4. Download latest release archive
TARBALL_NAME="${PROJECT}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${TARBALL_NAME}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "⬇️ Downloading ${TARBALL_NAME} from GitHub Releases..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL_NAME}"
elif command -v wget >/dev/null 2>&1; then
  wget -q "${DOWNLOAD_URL}" -O "${TMP_DIR}/${TARBALL_NAME}"
else
  echo "❌ Error: Neither curl nor wget is installed."
  exit 1
fi

echo "📦 Extracting binary..."
tar -xzf "${TMP_DIR}/${TARBALL_NAME}" -C "${TMP_DIR}"

echo "🚀 Installing to ${INSTALL_DIR}/${PROJECT}..."
if [ "${USE_SUDO}" = true ]; then
  sudo mv "${TMP_DIR}/${PROJECT}" "${INSTALL_DIR}/${PROJECT}"
  sudo chmod +x "${INSTALL_DIR}/${PROJECT}"
else
  mv "${TMP_DIR}/${PROJECT}" "${INSTALL_DIR}/${PROJECT}"
  chmod +x "${INSTALL_DIR}/${PROJECT}"
fi

# 5. Verify installation
if command -v "${PROJECT}" >/dev/null 2>&1; then
  echo "✅ TestGuard successfully installed!"
  "${PROJECT}" --version
else
  echo "⚠️ Installed to ${INSTALL_DIR}/${PROJECT}, but it is not currently on your PATH."
  echo "👉 Add this to your shell profile (~/.zshrc, ~/.bashrc):"
  echo "   export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
