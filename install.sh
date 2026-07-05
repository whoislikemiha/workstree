#!/bin/sh
# Install workstree from GitHub releases. No sudo, no dependencies beyond
# curl/tar. Installs to ~/.local/bin (override with WORKSTREE_INSTALL_DIR).
set -eu

REPO="whoislikemiha/workstree"
INSTALL_DIR="${WORKSTREE_INSTALL_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux | darwin) ;;
  *)
    echo "workstree: unsupported OS: $os (linux and darwin only)" >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *)
    echo "workstree: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

url="https://github.com/$REPO/releases/latest/download/workstree_${os}_${arch}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $url"
curl -fsSL "$url" -o "$tmp/workstree.tar.gz"
tar -xzf "$tmp/workstree.tar.gz" -C "$tmp" workstree

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/workstree" "$INSTALL_DIR/workstree"

echo "Installed $("$INSTALL_DIR/workstree" --version) to $INSTALL_DIR/workstree"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "NOTE: $INSTALL_DIR is not on your PATH" >&2 ;;
esac
