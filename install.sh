#!/usr/bin/env bash
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_DEPS="1"

usage() {
  cat <<'EOF'
Install ai-dev-setup.

Usage:
  ./install.sh [--no-deps]

Options:
  --no-deps    Only install the CLI wrapper. By default, the installer also
               bootstraps required dependencies such as Node.js/npm.
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
SOURCE="$SCRIPT_DIR/bin/ai-dev-setup"
TARGET="$INSTALL_DIR/ai-dev-setup"

[ -f "$SOURCE" ] || {
  printf 'error: missing %s\n' "$SOURCE" >&2
  exit 1
}

mkdir -p "$INSTALL_DIR"
cp "$SOURCE" "$TARGET"
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
