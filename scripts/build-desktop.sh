#!/bin/bash
set -euo pipefail

# Native download artifacts; GoReleaser separately builds the portable CLI archives.
VERSION="${1:?Usage: $0 <version> <output-directory>}"
OUTPUT="${2:?}"
if [ "$(uname -s)" != "Darwin" ]; then
  echo "The universal macOS desktop build requires macOS and Xcode command line tools." >&2
  exit 1
fi
mkdir -p "$OUTPUT/native" "$OUTPUT/direct"
for ARCH in amd64 arm64; do
  GOOS=darwin GOARCH="$ARCH" CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=12.0 \
    CGO_CFLAGS="-O2 -g -mmacosx-version-min=12.0" \
    CGO_LDFLAGS="-O2 -g -mmacosx-version-min=12.0" \
    go build -tags desktop,production \
    -ldflags "-s -w -X main.Version=${VERSION}" \
    -o "$OUTPUT/native/sushiro-$ARCH" .
done
lipo -create "$OUTPUT/native/sushiro-amd64" "$OUTPUT/native/sushiro-arm64" \
  -output "$OUTPUT/native/sushiro"
lipo "$OUTPUT/native/sushiro" -verify_arch x86_64 arm64

for ARCH in amd64 arm64; do
  GOOS=windows GOARCH="$ARCH" CGO_ENABLED=0 \
    go build -tags desktop,production,wv2runtime.browser \
    -ldflags "-s -w -H windowsgui -X main.Version=${VERSION}" \
    -o "$OUTPUT/direct/Sushiro-Overdose-${VERSION}-windows-${ARCH}.exe" .
done
