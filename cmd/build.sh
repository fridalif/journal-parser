#!/bin/bash

read -p "Enter version: " VERSION
mkdir build_$VERSION

echo "Bulding Linux_amd64"
GOOS=linux GOARCH=amd64 go build -o build_$VERSION/jp_linux_amd64 

echo "Bulding Windows_amd64"
GOOS=windows GOARCH=amd64 go build -o build_$VERSION/jp_windows_amd64.exe

echo "Building Linux_arm64"
GOOS=linux GOARCH=arm64 go build -o build_$VERSION/jp_linux_arm64

echo "Building Windows_arm64"
GOOS=windows GOARCH=arm64 go build -o build_$VERSION/jp_windows_arm64.exe

echo "Building Windows_386"
GOOS=windows GOARCH=386 go build -o build_$VERSION/jp_windows_386.exe

echo "Building Linux_386"
GOOS=linux GOARCH=386 go build -o build_$VERSION/jp_linux_386

echo "Building Darwin_amd64"
GOOS=darwin GOARCH=amd64 go build -o build_$VERSION/jp_darwin_amd64

echo "Building Darwin_arm64"
GOOS=darwin GOARCH=arm64 go build -o build_$VERSION/jp_darwin_arm64

echo "Done"