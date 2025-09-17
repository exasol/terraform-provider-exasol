.PHONY: build test fmt vet

build:
	go build -o bin/terraform-provider-exasol

test:
	go test ./... -v

fmt:
	gofmt -s -w .

vet:
	go vet ./...

# --------------------------------------------------------------------
# Install the provider binary into the local Terraform plugin cache
# so 'terraform init' can find it with source = "local/exasol".
# --------------------------------------------------------------------
install-local: build
	@os=$(shell go env GOOS) ; arch=$(shell go env GOARCH) ; \
	echo "Installing provider to ~/.terraform.d/plugins/local/exasol/0.1.0/$$os\_$$arch" ; \
	mkdir -p ~/.terraform.d/plugins/local/exasol/0.1.0/$$os\_$$arch ; \
	cp bin/terraform-provider-exasol \
	   ~/.terraform.d/plugins/local/exasol/0.1.0/$$os\_$$arch/terraform-provider-exasol_v0.1.0