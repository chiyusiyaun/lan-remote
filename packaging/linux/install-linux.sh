#!/usr/bin/env bash
# Install LAN Remote client/server binaries + desktop icons on Linux.
# Usage: sudo ./install-linux.sh [dir]
set -euo pipefail

PREFIX="${1:-/usr/local}"
BIN_DIR="$PREFIX/bin"
APP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/256x256/apps"

SRC="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SRC/../.." && pwd)"

need_root() {
  if [ ! -w "$BIN_DIR" ]; then
    echo "Need write access to $BIN_DIR — re-run with sudo, or pass a user dir:"
    echo "  $0 \$HOME/.local"
    exit 1
  fi
}

install_bins() {
  need_root
  install -d "$BIN_DIR"
  for name in lan-remote-client lan-remote-server; do
    if [ -f "$ROOT/dist/$name" ]; then
      install -m 0755 "$ROOT/dist/$name" "$BIN_DIR/$name"
      echo "installed $BIN_DIR/$name"
    elif [ -f "$ROOT/dist/${name}-linux" ]; then
      install -m 0755 "$ROOT/dist/${name}-linux" "$BIN_DIR/$name"
      echo "installed $BIN_DIR/$name (from ${name}-linux)"
    else
      echo "skip $name (binary not found in dist/)"
    fi
  done
}

# Convert ICO (or use embedded PNG) via Python/Pillow if available; else copy SVG-less PNG fallback.
install_icons() {
  install -d "$ICON_DIR" "$APP_DIR"
  python3 - <<'PY' || true
import os, sys
try:
    from PIL import Image
except ImportError:
    sys.exit(0)
root = os.environ.get("SRC_ROOT") or "."
# prefer generated PNGs next to this script
pairs = [
    ("icon-client.png", "lan-remote-client.png"),
    ("icon-server.png", "lan-remote-server.png"),
]
icon_dir = os.path.expanduser("~/.local/share/icons/hicolor/256x256/apps")
os.makedirs(icon_dir, exist_ok=True)
for src, dst in pairs:
    p = os.path.join(root, "packaging", "linux", src)
    if os.path.isfile(p):
        im = Image.open(p).convert("RGBA")
        im = im.resize((256, 256), Image.LANCZOS)
        im.save(os.path.join(icon_dir, dst))
        print("icon", dst)
PY
  # copy desktop files
  install -m 0644 "$SRC/lan-remote-client.desktop" "$APP_DIR/"
  install -m 0644 "$SRC/lan-remote-server.desktop" "$APP_DIR/"
  echo "desktop entries installed to $APP_DIR"
}

export SRC_ROOT="$ROOT"
install_bins
install_icons

# update icon cache if tool exists
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
fi

echo "Done. Launch from app menu: LAN Remote Client / Server"
