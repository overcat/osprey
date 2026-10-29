#!/bin/sh
# Builds osp.exe and osprey.com, the console programs that run Osprey.exe
# from a terminal (see cmd/osp), into resources/windows-<arch>/, which
# mygo ships beside Osprey.exe. Run it before building Osprey for Windows.
set -eu

cd "$(dirname "$0")/.."
for arch in amd64 arm64; do
	out="resources/windows-$arch"
	mkdir -p "$out"
	GOOS=windows GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$out/osp.exe" ./cmd/osp
	cp "$out/osp.exe" "$out/osprey.com"
done
