.PHONY: help build clean test install fmt vet deps dev

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the Terraform provider binary
	export GOROOT=/opt/homebrew/opt/go/libexec && GOOS=darwin GOARCH=arm64 go build -o terraform-provider-serverscomx

clean: ## Clean build artifacts
	rm -f terraform-provider-serverscomx
	go clean -cache

test: ## Run tests
	export GOROOT=/opt/homebrew/opt/go/libexec && go test -v ./...

install: build ## Build and install the provider locally
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/adconnectdevops/serverscomx/0.1.0/darwin_arm64/
	cp terraform-provider-serverscomx ~/.terraform.d/plugins/registry.terraform.io/adconnectdevops/serverscomx/0.1.0/darwin_arm64/

fmt: ## Format Go code (matches CI: gofmt -s for simplifications)
	export GOROOT=/opt/homebrew/opt/go/libexec && gofmt -s -w .

vet: ## Run go vet
	export GOROOT=/opt/homebrew/opt/go/libexec && go vet ./...

deps: ## Download dependencies
	export GOROOT=/opt/homebrew/opt/go/libexec && go mod tidy
	export GOROOT=/opt/homebrew/opt/go/libexec && go mod download

dev: ## Run in development mode with debug
	export GOROOT=/opt/homebrew/opt/go/libexec && go run . -debug
