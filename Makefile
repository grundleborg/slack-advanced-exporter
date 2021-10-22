.PHONY: build build-linux build-mac build-mac-arm build-windows release clean

ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

build: ## Build Slack Advanced Exporter for the current platform and architecture
	go build .

build-linux: ## Build Slack Advanced Exporter for Linux
	@mkdir -p ${ROOT}build
	cd ${ROOT}build && GOOS=linux GOARCH=amd64 go build ..

build-mac: ## Build Slack Advanced Exporter for Mac
	@mkdir -p ${ROOT}build
	cd ${ROOT}build && GOOS=darwin GOARCH=amd64 go build ..

build-mac-arm: ## Build Slack Advanced Exporter for ARM Mac
	@mkdir -p ${ROOT}build
	cd ${ROOT}build && GOOS=darwin GOARCH=arm64 go build ..

build-windows: ## Build Slack Advanced Exporter for Windows
	@mkdir -p ${ROOT}build
	cd ${ROOT}build && GOOS=windows GOARCH=amd64 go build ..

release: clean ## Build and package the release artefacts
	@mkdir -p ${ROOT}build
	@mkdir -p ${ROOT}release
	cd ${ROOT}build && GOOS=linux GOARCH=amd64 go build ..
	cd ${ROOT}build && tar -czf ../release/slack-advanced-exporter.linux-amd64.tar.gz slack-advanced-exporter
	cd ${ROOT}build && GOOS=darwin GOARCH=amd64 go build ..
	cd ${ROOT}build && tar -czf ../release/slack-advanced-exporter.darwin-amd64.tar.gz slack-advanced-exporter
	cd ${ROOT}build && GOOS=darwin GOARCH=arm64 go build ..
	cd ${ROOT}build && tar -czf ../release/slack-advanced-exporter.darwin-arm64.tar.gz slack-advanced-exporter
	cd ${ROOT}build && GOOS=windows GOARCH=amd64 go build ..
	cd ${ROOT}build && zip -q ../release/slack-advanced-exporter.windows-amd64.zip slack-advanced-exporter.exe
	cd ${ROOT}release && sha256sum ./slack-advanced-exporter.*

clean: ## Remove all build and release artefacts
	rm -rf build
	rm -rf release
	rm -rf slack-advanced-exporter
	rm -rf slack-advanced-exporter.exe
