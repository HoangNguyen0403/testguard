# 🛡️ TestGuard Makefile — Development & Quick Release Automation

BINARY_NAME := testguard
MODULE      := github.com/HoangNguyen0403/testguard
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: all build test lint format clean install run doctor release-patch release-minor release-major release-dry tag-delete help

# Default target
all: build test

## 🔨 Build & Local Development

build: ## Build the native testguard binary into ./bin/
	@echo "📦 Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/testguard
	@echo "✅ Binary created at bin/$(BINARY_NAME)"

install: ## Install binary to $GOPATH/bin or /usr/local/bin
	@echo "🚀 Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	go install -ldflags "$(LDFLAGS)" ./cmd/testguard
	@echo "✅ Installed! Run 'testguard --version' to verify."

run: build ## Run testguard locally
	@./bin/$(BINARY_NAME)

test: ## Run unit and integration tests with race detector
	@echo "🧪 Running test suite..."
	go test -v -race ./...

lint: ## Run go vet and static checks
	@echo "🔍 Running code checks..."
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	fi

format: ## Format all Go code
	@echo "🎨 Formatting source files..."
	go fmt ./...

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning artifacts..."
	@rm -rf bin dist .testguard coverage.out
	@echo "✅ Clean complete."

doctor: ## Run agent-skills-standard health audit
	@if command -v agent-skills-standard >/dev/null 2>&1; then \
		agent-skills-standard doctor; \
	fi

## 🚀 Quick Release Automation

# Helper: extracts current latest semantic tag or defaults to v0.1.0
CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")

release-dry: ## Run GoReleaser in dry-run snapshot mode without publishing
	@echo "🔍 Running release dry-run snapshot..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "⚠️ GoReleaser not installed locally. Simulating multi-platform build:"; \
		GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/testguard; \
		GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/testguard; \
		GOOS=linux  GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64  ./cmd/testguard; \
		GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/testguard; \
		echo "✅ Multi-platform binaries compiled successfully in ./bin/"; \
	fi

release-patch: ## Bump patch version, tag, and push to GitHub (v0.1.0 -> v0.1.1)
	@echo "Current tag: $(CURRENT_TAG)"
	@bash scripts/bump-tag.sh patch

release-minor: ## Bump minor version, tag, and push to GitHub (v0.1.0 -> v0.2.0)
	@echo "Current tag: $(CURRENT_TAG)"
	@bash scripts/bump-tag.sh minor

release-major: ## Bump major version, tag, and push to GitHub (v0.1.0 -> v1.0.0)
	@echo "Current tag: $(CURRENT_TAG)"
	@bash scripts/bump-tag.sh major

release: ## Release with explicit custom tag: make release TAG=v0.2.0
	@if [ -z "$(TAG)" ]; then \
		echo "❌ Error: TAG is required. Usage: make release TAG=v0.2.0"; \
		exit 1; \
	fi
	@echo "🚀 Tagging release $(TAG)..."
	git tag $(TAG)
	@echo "⬆️ Pushing tag $(TAG) to origin..."
	git push origin $(TAG)
	@echo "🎉 Tag $(TAG) pushed! GitHub Actions release workflow triggered."

tag-delete: ## Delete a release tag locally and remotely: make tag-delete TAG=v0.2.0
	@if [ -z "$(TAG)" ]; then \
		echo "❌ Error: TAG is required. Usage: make tag-delete TAG=v0.2.0"; \
		exit 1; \
	fi
	@echo "⚠️ Deleting tag $(TAG)..."
	git tag -d $(TAG) || true
	git push --delete origin $(TAG) || true
	@echo "✅ Tag $(TAG) deleted."

## 📖 Help

help: ## Show this help message
	@echo "TestGuard CLI — Available Make Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
