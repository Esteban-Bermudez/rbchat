#!/bin/sh
set -eu

REPO="Esteban-Bermudez/rbchat"
BIN="rbchat"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

detect_os_arch() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
  esac

  case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
  esac
}

fetch_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl -sfL"
  elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget -qO-"
  else
    echo "Need curl or wget to install."
    exit 1
  fi

  VERSION=$($DOWNLOADER "https://api.github.com/repos/$REPO/releases/latest" |
    grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)",/\1/')

  if [ -z "$VERSION" ]; then
    echo "Could not fetch latest version."
    exit 1
  fi
}

download_and_install() {
  EXT="tar.gz"
  case "$OS" in
    windows) EXT="zip" ;;
  esac

  ARCHIVE="${BIN}_${OS}_${ARCH}.${EXT}"
  URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
  CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

  TMPDIR=$(mktemp -d)
  cd "$TMPDIR"

  echo "Downloading $BIN $VERSION for $OS/$ARCH..."

  $DOWNLOADER "$URL" > "$ARCHIVE"
  $DOWNLOADER "$CHECKSUM_URL" > checksums.txt 2>/dev/null || true

  if [ -f checksums.txt ]; then
    GREP_CMD="grep"
    if command -v sha256sum >/dev/null 2>&1; then
      CHECK_CMD="sha256sum -c"
      EXPECTED=$(grep "$ARCHIVE" checksums.txt | cut -d' ' -f1)
      COMPUTED=$(sha256sum "$ARCHIVE" | cut -d' ' -f1)
      [ "$EXPECTED" = "$COMPUTED" ] || echo "Warning: checksum mismatch"
    elif command -v shasum >/dev/null 2>&1; then
      CHECK_CMD="shasum -a 256 -c"
      EXPECTED=$(grep "$ARCHIVE" checksums.txt | cut -d' ' -f1)
      COMPUTED=$(shasum -a 256 "$ARCHIVE" | cut -d' ' -f1)
      [ "$EXPECTED" = "$COMPUTED" ] || echo "Warning: checksum mismatch"
    fi
  fi

  case "$OS" in
    windows)
      unzip -q "$ARCHIVE"
      mv "${BIN}.exe" "$BIN"
      echo "Extracted $BIN.exe — move it somewhere in your PATH."
      ;;
    *)
      tar -xzf "$ARCHIVE"
      chmod +x "$BIN"
      ;;
  esac

  if [ -f "$BIN" ]; then
    mkdir -p "$INSTALL_DIR"
    mv "$BIN" "$INSTALL_DIR/$BIN"
    echo "Installed $BIN to $INSTALL_DIR/$BIN"
  fi

  rm -rf "$TMPDIR"
}

detect_os_arch
fetch_latest_version
download_and_install

echo "Run '$BIN' to start."
