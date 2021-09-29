#!/bin/bash

rm ./slack-advanced-exporter ./slack-advanced-exporter.darwin-amd64.tar.gz ./slack-advanced-exporter.linux-amd64.tar.gz ./slack-advanced-exporter.windows-amd64.zip

echo "Building macOS"
env GOOS=darwin GOARCH=amd64 go build .
tar -czf slack-advanced-exporter.darwin-amd64.tar.gz slack-advanced-exporter
rm ./slack-advanced-exporter

echo "Building Linux"
env GOOS=linux GOARCH=amd64 go build .
tar -czf slack-advanced-exporter.linux-amd64.tar.gz slack-advanced-exporter
rm ./slack-advanced-exporter

echo "Building Windows"
env GOOS=windows GOARCH=amd64 go build .
zip -q slack-advanced-exporter.windows-amd64.zip slack-advanced-exporter.exe
rm ./slack-advanced-exporter.exe

sha256sum ./slack-advanced-exporter.*