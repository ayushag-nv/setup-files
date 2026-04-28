#!/usr/bin/env bash
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_DEPS="1"
BOOTSTRAP_GO_VERSION="${BOOTSTRAP_GO_VERSION:-1.25.0}"

usage() {
  cat <<'EOF'
Install wolfpack.

Usage:
  ./install.sh [--no-deps]

Options:
  --no-deps    Only build and install the CLI binary. By default, the
               installer also bootstraps runtime dependencies and developer tools.
EOF
}

case "${1:-}" in
  "")
    ;;
  --no-deps)
    INSTALL_DEPS="0"
    ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$INSTALL_DIR/wolfpack"
TMP_GO_DIR=""

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

fetch_url() {
  local url="$1"
  local output="$2"

  if have_cmd curl; then
    curl -fsSL "$url" -o "$output"
    return
  fi

  if have_cmd wget; then
    wget -qO "$output" "$url"
    return
  fi

  printf 'error: curl or wget is required to download the temporary Go toolchain\n' >&2
  exit 1
}

bootstrap_go() {
  local os arch url archive

  if have_cmd go; then
    command -v go
    return
  fi

  case "$(uname -s)" in
    Linux) os="linux" ;;
    Darwin) os="darwin" ;;
    *)
      printf 'error: unsupported OS: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      printf 'error: unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac

  TMP_GO_DIR="$(mktemp -d "${TMPDIR:-/tmp}/wolfpack-go.XXXXXX")"
  archive="$TMP_GO_DIR/go.tar.gz"
  url="https://go.dev/dl/go${BOOTSTRAP_GO_VERSION}.${os}-${arch}.tar.gz"

  printf 'Go was not found; downloading temporary Go %s from %s\n' "$BOOTSTRAP_GO_VERSION" "$url" >&2
  fetch_url "$url" "$archive"
  tar -C "$TMP_GO_DIR" -xzf "$archive"
  printf '%s/go/bin/go\n' "$TMP_GO_DIR"
}

cleanup() {
  if [ -n "$TMP_GO_DIR" ]; then
    rm -rf "$TMP_GO_DIR"
  fi
}
trap cleanup EXIT

GO_BIN="$(bootstrap_go)"

mkdir -p "$INSTALL_DIR"
"$GO_BIN" build -trimpath -ldflags "-s -w" -o "$TARGET" "$SCRIPT_DIR/cmd/wolfpack"
chmod +x "$TARGET"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    BASHRC="$HOME/.bashrc"
    touch "$BASHRC"
    if ! grep -Fq "$INSTALL_DIR" "$BASHRC"; then
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >>"$BASHRC"
      printf 'Added %s to PATH in %s\n' "$INSTALL_DIR" "$BASHRC"
    fi
    ;;
esac

if [ "$(uname -s)" = "Darwin" ]; then
  BASH_PROFILE="$HOME/.bash_profile"
  touch "$BASH_PROFILE"
  if ! grep -Fq '.bashrc' "$BASH_PROFILE"; then
    printf '\n[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"\n' >>"$BASH_PROFILE"
  fi
fi

printf 'Installed %s\n' "$TARGET"

if [ "$INSTALL_DEPS" = "1" ]; then
  "$TARGET" deps
fi
