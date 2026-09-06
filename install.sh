#!/bin/sh
set -eu

repo="bpstr/rootmark"
install_dir="${ROOTMARK_INSTALL_DIR:-/usr/local/bin}"
version="${ROOTMARK_VERSION:-}"

if [ -z "$version" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [ -z "$version" ]; then
  echo "rootmark: could not determine the latest release" >&2
  exit 1
fi

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "rootmark: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "rootmark: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

plain_version="${version#v}"
archive="rootmark_${plain_version}_${os}_${arch}.tar.gz"
url="https://github.com/$repo/releases/download/$version/$archive"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

curl -fsSL "$url" -o "$tmp/$archive"
tar -xzf "$tmp/$archive" -C "$tmp"

if [ -d "$install_dir" ] && [ -w "$install_dir" ]; then
  install -m 0755 "$tmp/rootmark" "$install_dir/rootmark"
elif [ ! -e "$install_dir" ] && mkdir -p "$install_dir" 2>/dev/null; then
  install -m 0755 "$tmp/rootmark" "$install_dir/rootmark"
elif command -v sudo >/dev/null 2>&1; then
  sudo mkdir -p "$install_dir"
  sudo install -m 0755 "$tmp/rootmark" "$install_dir/rootmark"
else
  echo "rootmark: cannot write to $install_dir; set ROOTMARK_INSTALL_DIR to a writable directory" >&2
  exit 1
fi

echo "Installed rootmark $version to $install_dir/rootmark"
