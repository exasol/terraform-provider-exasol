.PHONY: build test fmt vet clean install-local docs lint help

# Variables
BINARY_NAME=terraform-provider-exasol
PROVIDER_NAME=exasol/bi-terraform-provider-exasol
VERSION?=0.1.0

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the provider binary
	@echo "Building $(BINARY_NAME)..."
	@go build -ldflags="-X main.version=$(VERSION)" -o bin/$(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	@go test ./... -v -cover

test-integration: ## Run integration tests (requires database)
	@echo "Running integration tests..."
	@go test ./... -v -tags=integration

fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -s -w .
	@go mod tidy

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/
	@go clean

# Install the provider binary into the local Terraform plugin cache
install-local: build ## Install provider locally for testing
	@os=$$(go env GOOS) ; arch=$$(go env GOARCH) ; \
	echo "Installing provider to ~/.terraform.d/plugins/local/$(PROVIDER_NAME)/$(VERSION)/$$os\_$$arch" ; \
	mkdir -p ~/.terraform.d/plugins/local/$(PROVIDER_NAME)/$(VERSION)/$$os\_$$arch ; \
	cp bin/$(BINARY_NAME) \
	   ~/.terraform.d/plugins/local/$(PROVIDER_NAME)/$(VERSION)/$$os\_$$arch/$(BINARY_NAME)_v$(VERSION)

release: ## Build release binaries
	@echo "Building release binaries..."
	@goreleaser build --snapshot --rm-dist

check: fmt vet lint test ## Run all checks

all: clean check build ## Run all steps