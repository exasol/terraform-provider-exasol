terraform {
  required_providers {
    exasol = {
      source  = "local/exasol/terraform-provider-exasol"
      version = "0.2.0"
    }
  }
}

provider "exasol" {
  host     = var.exa_host
  port     = 8563
  user     = var.exa_user
  password = var.exa_password
}

# Example resources to grant privileges on
resource "exasol_schema" "example" {
  name = "ANALYTICS"
}

resource "exasol_role" "analyst" {
  name = "ANALYST_ROLE"
}

resource "exasol_user" "analyst_user" {
  name      = "ANALYST_USER"
  auth_type = "PASSWORD"
  password  = "SecurePassword123"
}

# =============================================================================
# ROLE GRANTS - Granting a role to a user
# =============================================================================

# Method 1: Role grant using SYSTEM privilege_type with object_type="ROLE"
resource "exasol_grant" "role_to_user_method1" {
  grantee_name      = exasol_user.analyst_user.name
  privilege_type    = "SYSTEM"
  privilege         = exasol_role.analyst.name
  object_type       = "ROLE"
  with_admin_option = false
}

# Method 2: Role grant using OBJECT privilege_type with object_type="ROLE"
# (Alternative syntax - both work the same way)
resource "exasol_grant" "role_to_user_method2" {
  grantee_name   = "ANOTHER_USER"
  privilege_type = "OBJECT"
  privilege      = exasol_role.analyst.name
  object_type    = "ROLE"
  object_name    = exasol_role.analyst.name
}

# =============================================================================
# SYSTEM PRIVILEGES - System-level privileges
# =============================================================================

# Grant system privilege to a role
resource "exasol_grant" "create_session" {
  grantee_name      = exasol_role.analyst.name
  privilege_type    = "SYSTEM"
  privilege         = "CREATE SESSION"
  with_admin_option = false
}

# Grant system privilege with admin option
resource "exasol_grant" "use_any_schema" {
  grantee_name      = exasol_role.analyst.name
  privilege_type    = "SYSTEM"
  privilege         = "USE ANY SCHEMA"
  with_admin_option = true
}

# Grant CREATE TABLE system privilege
resource "exasol_grant" "create_table" {
  grantee_name   = exasol_role.analyst.name
  privilege_type = "SYSTEM"
  privilege      = "CREATE TABLE"
}

# =============================================================================
# OBJECT PRIVILEGES - Schema-level privileges
# =============================================================================

# Grant USAGE on a schema to a role
resource "exasol_grant" "schema_usage" {
  grantee_name   = exasol_role.analyst.name
  privilege_type = "OBJECT"
  privilege      = "USAGE"
  object_type    = "SCHEMA"
  object_name    = exasol_schema.example.name
}

# Grant CREATE TABLE on a schema
resource "exasol_grant" "schema_create_table" {
  grantee_name   = exasol_role.analyst.name
  privilege_type = "OBJECT"
  privilege      = "CREATE TABLE"
  object_type    = "SCHEMA"
  object_name    = exasol_schema.example.name
}

# Grant ALL privileges on a schema
resource "exasol_grant" "schema_all" {
  grantee_name   = "DBA_ROLE"
  privilege_type = "OBJECT"
  privilege      = "ALL"
  object_type    = "SCHEMA"
  object_name    = exasol_schema.example.name
}

# =============================================================================
# OBJECT PRIVILEGES - Table-level privileges
# =============================================================================

# Grant SELECT on a table
resource "exasol_grant" "table_select" {
  grantee_name   = exasol_role.analyst.name
  privilege_type = "OBJECT"
  privilege      = "SELECT"
  object_type    = "TABLE"
  object_name    = "ANALYTICS.SALES_DATA"
}

# Grant INSERT and UPDATE on a table
resource "exasol_grant" "table_insert" {
  grantee_name   = "DATA_LOADER_ROLE"
  privilege_type = "OBJECT"
  privilege      = "INSERT"
  object_type    = "TABLE"
  object_name    = "ANALYTICS.SALES_DATA"
}

# Grant ALL on a table
resource "exasol_grant" "table_all" {
  grantee_name   = "TABLE_OWNER"
  privilege_type = "OBJECT"
  privilege      = "ALL"
  object_type    = "TABLE"
  object_name    = "ANALYTICS.SALES_DATA"
}

# =============================================================================
# OBJECT PRIVILEGES - View-level privileges
# =============================================================================

# Grant SELECT on a view
resource "exasol_grant" "view_select" {
  grantee_name   = exasol_role.analyst.name
  privilege_type = "OBJECT"
  privilege      = "SELECT"
  object_type    = "VIEW"
  object_name    = "ANALYTICS.SALES_SUMMARY"
}