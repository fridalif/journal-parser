#!/bin/bash

read -p "Enter version: " VERSION
mkdir -p build_$VERSION

echo "Building Linux_amd64"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build_$VERSION/jp_linux_amd64

echo "Building Windows_amd64"
CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu" GOOS=windows GOARCH=amd64 go build -o build_$VERSION/jp_windows_amd64.exe

echo "Building Linux_arm64"
CGO_ENABLED=1 CC="zig cc -target aarch64-linux-gnu" GOOS=linux GOARCH=arm64 go build -o build_$VERSION/jp_linux_arm64

echo "Building Windows_arm64"
CGO_ENABLED=1 CC="zig cc -target aarch64-windows-gnu" GOOS=windows GOARCH=arm64 go build -o build_$VERSION/jp_windows_arm64.exe

echo "Building Windows_386"
CGO_ENABLED=1 CC="zig cc -target x86-windows-gnu" GOOS=windows GOARCH=386 go build -o build_$VERSION/jp_windows_386.exe

echo "Building Linux_386"
CGO_ENABLED=1 CC="zig cc -target x86-linux-gnu" GOOS=linux GOARCH=386 go build -o build_$VERSION/jp_linux_386

echo "Done"