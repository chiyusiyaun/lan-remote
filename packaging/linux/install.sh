#!/bin/sh
# Install LAN Remote (client + server) system-wide.
# Run from the extracted package directory (contains binaries).
set -e
DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PREFIX="${PREFIX:-/usr/local}"

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root: sudo $0"
  exit 1
fi

install -d "$PREFIX/bin"
install -d /usr/share/applications
install -d /usr/share/icons/hicolor/256x256/apps

if [ -f "$DIR/lan-remote-client-linux" ]; then
  install -m 0755 "$DIR/lan-remote-client-linux" "$PREFIX/bin/lan-remote-client"
elif [ -f "$DIR/lan-remote-client" ]; then
  install -m 0755 "$DIR/lan-remote-client" "$PREFIX/bin/lan-remote-client"
fi

if [ -f "$DIR/lan-remote-server-linux" ]; then
  install -m 0755 "$DIR/lan-remote-server-linux" "$PREFIX/bin/lan-remote-server"
elif [ -f "$DIR/lan-remote-server" ]; then
  install -m 0755 "$DIR/lan-remote-server" "$PREFIX/bin/lan-remote-server"
fi

[ -f "$DIR/lan-remote-client.desktop" ] && install -m 0644 "$DIR/lan-remote-client.desktop" /usr/share/applications/
[ -f "$DIR/lan-remote-server.desktop" ] && install -m 0644 "$DIR/lan-remote-server.desktop" /usr/share/applications/
[ -f "$DIR/icon-client.png" ] && install -m 0644 "$DIR/icon-client.png" /usr/share/icons/hicolor/256x256/apps/lan-remote-client.png
[ -f "$DIR/icon-server.png" ] && install -m 0644 "$DIR/icon-server.png" /usr/share/icons/hicolor/256x256/apps/lan-remote-server.png

command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database /usr/share/applications 2>/dev/null || true

echo "Installed:"
echo "  $PREFIX/bin/lan-remote-client"
echo "  $PREFIX/bin/lan-remote-server"
echo "Start: lan-remote-server   /   lan-remote-client"
