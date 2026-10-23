#!/bin/sh
# Builds Accolade's web version into dist/web (or $1): the WebAssembly,
# Go's wasm_exec.js and the page in web/. Serve the folder over http(s).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
out=${1:-$root/dist/web}
mkdir -p "$out"
GOOS=js GOARCH=wasm GOWORK=off go build -C "$root" -ldflags="-s -w -X main.version=$(git -C "$root" describe --tags --always)" -o "$out/accolade.wasm" .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/"
cp -r "$root"/web/. "$out/"
echo "web version in $out ($(du -h "$out/accolade.wasm" | cut -f1) of WebAssembly)"
