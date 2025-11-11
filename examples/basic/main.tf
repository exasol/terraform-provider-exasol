terraform {
  required_providers {
    exasol = {
      source  = "local/exasol/terraform-provider-exasol"
      version = "~> 0.1.6"
    }
  }
}

provider "exasol" {
  host     = var.exa_host
  port     = 8563
  user     = var.exa_user
  password = var.exa_password
}

# Create a user
resource "exasol_user" "example_user" {
  name      = "EXAMPLE_USER"
  auth_type = "PASSWORD"
  password  = "SecurePassword123!"
}

# Create a role
resource "exasol_role" "analyst" {
  name = "ANALYST"
}

# Create a schema
resource "exasol_schema" "curated" {
  name = "CURATED"
}

# Grant system privilege - CREATE SESSION (required for login)
resource "exasol_system_privilege" "user_login" {
  grantee   = exasol_user.example_user.name
  privilege = "CREATE SESSION"
}

# Grant system privilege to role
resource "exasol_system_privilege" "use_any_schema" {
  grantee   = exasol_role.analyst.name
  privilege = "USE ANY SCHEMA"
}

# Grant object privileges on schema
resource "exasol_object_privilege" "schema_usage" {
  grantee     = exasol_role.analyst.name
  privileges  = ["USAGE", "SELECT"]
  object_type = "SCHEMA"
  object_name = exasol_schema.curated.name
}

# Grant SELECT on table
resource "exasol_object_privilege" "select_table" {
  grantee     = exasol_role.analyst.name
  privileges  = ["SELECT"]
  object_type = "TABLE"
  object_name = "CURATED.CUSTOMERS"
}

# Grant role to user
resource "exasol_role_grant" "analyst_to_user" {
  role    = exasol_role.analyst.name
  grantee = exasol_user.example_user.name
}
