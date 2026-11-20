#!/bin/sh
# Builds Accolade's web version into dist/web (or $1): the WebAssembly,
# Go's wasm_exec.js, the page in web/ and the help pages in help/. Serve the folder over http(s).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
out=${1:-$root/dist/web}
mkdir -p "$out"
version=$(git -C "$root" describe --tags --always)
GOOS=js GOARCH=wasm GOWORK=off go build -C "$root" -ldflags="-s -w -X main.version=$version" -o "$out/accolade.wasm" .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/"
cp -r "$root"/web/. "$out/"
# the help as web pages, /help/<page>.html (docs/help; pure Go, no cgo)
CGO_ENABLED=0 GOWORK=off go run -C "$root" ./docs/help/cmd/helpsite -version "$version" "$out/help"
echo "web version in $out ($(du -h "$out/accolade.wasm" | cut -f1) of WebAssembly)"
