#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.toolchain}"
BIN_DIR="$ROOT/bin"
mkdir -p "$BIN_DIR"

# Find all sub-bin dirs (e.g., .toolchain/go/bin, .toolchain/cmake/bin, etc.)
while IFS= read -r -d '' d; do
  # Symlink each file into .toolchain/bin (skip if exists)
  find "$d" -maxdepth 1 -type f -perm -111 -print0 2>/dev/null | while IFS= read -r -d '' f; do
    lnname="$BIN_DIR/$(basename "$f")"
    if [ ! -e "$lnname" ]; then
      ln -s "$f" "$lnname" || true
    fi
  done
done < <(find "$ROOT" -type d -name bin -print0)
