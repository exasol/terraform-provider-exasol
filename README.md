# filepath: README.md
# Terraform Provider for Exasol

A Terraform provider for managing Exasol database resources.

## Features

- User management
- Role management
- Schema management
- Grant management

## Installation

### Using Terraform Registry

```hcl
terraform {
  required_providers {
    exasol = {
      source  = "exasol/bi-terraform-provider-exasol"
      version = "~> 0.1.0"
    }
  }
}
```

### Local Development

```bash
make install-local
```

## Usage

```hcl
provider "exasol" {
  dsn = "exa:localhost:8563"
  # ... other configuration
}

resource "exasol_user" "example" {
  name     = "testuser"
  password = "password123"
}
```

## Development

### Prerequisites

- Go 1.21+
- Terraform 1.0+
- Make

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Acceptance Testing

```bash
make test-acc
```
