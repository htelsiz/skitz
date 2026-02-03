#!/usr/bin/env bash
set -e

echo "Building skitz..."
go build -ldflags="-s -w" -o skitz .

echo "Installing to ~/.local/bin/skitz..."
mkdir -p ~/.local/bin
mv skitz ~/.local/bin/

echo "✓ Installed to ~/.local/bin/skitz"
