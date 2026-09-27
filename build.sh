#!/usr/bin/env sh
# Cross-compile SingaporePSI.exe (Windows x64) from Linux/macOS (or Git Bash on Windows).
# Requires Go (see go.mod) and network access the first time (modules + go-winres).
set -eu
cd "$(dirname "$0")"

GOWINRES="github.com/tc-hib/go-winres@v0.3.3"

echo "==> Generating Windows resources (icon, version info, manifest)"
go run "$GOWINRES" make --in winres/winres.json --arch amd64

echo "==> go vet (windows/amd64)"
GOOS=windows GOARCH=amd64 go vet ./...

echo "==> Building dist/SingaporePSI.exe"
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false \
  -ldflags "-H windowsgui -s -w" -o dist/SingaporePSI.exe .

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum dist/SingaporePSI.exe
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 dist/SingaporePSI.exe
fi
echo "Done: dist/SingaporePSI.exe"
