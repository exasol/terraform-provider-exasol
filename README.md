# filepath: README.md
# Terraform Provider for Exasol

A Terraform provider for managing Exasol database resources.

## Features

- **User management** - Create and manage database users with various authentication methods
- **Role management** - Define and manage database roles
- **Schema management** - Create and configure database schemas
- **Connection management** - Manage external connections (S3, FTP, JDBC, etc.)
- **Privilege management** - Three dedicated resources for clear privilege management:
  - `exasol_system_privilege` - System-level privileges (CREATE SESSION, CREATE TABLE, etc.)
  - `exasol_object_privilege` - Object-level privileges (SELECT, INSERT, etc. on tables/schemas/views)
  - `exasol_role_grant` - Grant roles to users or other roles

## Installation

### Using Terraform Registry

```hcl
terraform {
  required_providers {
    exasol = {
      source  = "registry.terraform.io/exasol/bi-terraform-provider-exasol"
      version = "~> 0.2.0"
    }
  }
}
```

### Local Development

For local development, use the `local` source:

```hcl
terraform {
  required_providers {
    exasol = {
      source  = "local/exasol/bi-terraform-provider-exasol"
      version = "0.2.0"
    }
  }
}
```

Then install the provider locally:

```bash
make install-local
```

## Usage

```hcl
provider "exasol" {
  host     = "localhost"
  port     = 8563
  user     = "sys"
  password = "exasol"
}

resource "exasol_user" "example" {
  name      = "testuser"
  auth_type = "PASSWORD"
  password  = "password123"
}

resource "exasol_connection" "s3" {
  name     = "MY_S3_BUCKET"
  to       = "https://my-bucket.s3.us-east-1.amazonaws.com"
  user     = "AWS_ACCESS_KEY"
  password = "AWS_SECRET_KEY"
}

resource "exasol_role" "analyst" {
  name = "ANALYST_ROLE"
}

# Grant system privilege
resource "exasol_system_privilege" "create_session" {
  grantee   = exasol_user.example.name
  privilege = "CREATE SESSION"
}

# Grant multiple object privileges (can be a single privilege or list)
resource "exasol_object_privilege" "table_access" {
  grantee     = exasol_role.analyst.name
  privileges  = ["SELECT", "INSERT", "UPDATE"]  # List of privileges
  object_type = "TABLE"
  object_name = "MYSCHEMA.MYTABLE"
}

# Grant role to user
resource "exasol_role_grant" "user_role" {
  role    = exasol_role.analyst.name
  grantee = exasol_user.example.name
}
```

## Resource Examples

See the [examples/](examples/) directory for complete examples of each resource type:
- [examples/privileges/](examples/privileges/) - System privileges, object privileges, and role grants
- [examples/connections/](examples/connections/) - Various connection types (S3, FTP, JDBC, etc.)
- [examples/basic/](examples/basic/) - Basic resource usage

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
