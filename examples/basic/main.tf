terraform {
  required_providers {
    exasol = {
      source  = "exasol/bi-terraform-provider-exasol"
      version = "0.1.0"
    }
  }
}

provider "exasol" {
  host     = var.exa_host
  port     = 8563
  user     = var.exa_user
  password = var.exa_password
}

# Grant USAGE on a schema to a role
resource "exasol_grant" "schema_usage" {
  grantee_type      = "ROLE"
  grantee_name      = "ANALYST"
  privilege_type    = "OBJECT"
  privilege         = "USAGE"
  object_type       = "SCHEMA"
  object_name       = "CURATED"
  with_grant_option = false
}

# System privilege to role
resource "exasol_grant" "use_any_schema" {
  grantee_type     = "ROLE"
  grantee_name     = "ANALYST"
  privilege_type   = "SYSTEM"
  privilege        = "USE ANY SCHEMA"
  with_admin_option = false
}

# Grant SELECT on table
resource "exasol_grant" "select_table" {
  grantee_type      = "ROLE"
  grantee_name      = "ANALYST"
  privilege_type    = "OBJECT"
  privilege         = "SELECT"
  object_type       = "TABLE"
  object_name       = "CURATED.CUSTOMERS"
}
