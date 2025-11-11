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

# Create example resources
resource "exasol_schema" "analytics" {
  name = "ANALYTICS"
}

resource "exasol_role" "analyst" {
  name = "ANALYST_ROLE"
}

resource "exasol_role" "data_loader" {
  name = "DATA_LOADER_ROLE"
}

resource "exasol_user" "john" {
  name      = "JOHN_DOE"
  auth_type = "PASSWORD"
  password  = "SecurePassword123"
}

resource "exasol_user" "jane" {
  name      = "JANE_SMITH"
  auth_type = "PASSWORD"
  password  = "SecurePassword456"
}

# =============================================================================
# SYSTEM PRIVILEGES - Using exasol_system_privilege
# =============================================================================

# Grant CREATE SESSION so user can log in
resource "exasol_system_privilege" "john_create_session" {
  grantee   = exasol_user.john.name
  privilege = "CREATE SESSION"
}

# Grant CREATE TABLE system privilege to a role
resource "exasol_system_privilege" "analyst_create_table" {
  grantee   = exasol_role.analyst.name
  privilege = "CREATE TABLE"
}

# Grant USE ANY SCHEMA with admin option
resource "exasol_system_privilege" "analyst_use_any_schema" {
  grantee           = exasol_role.analyst.name
  privilege         = "USE ANY SCHEMA"
  with_admin_option = true
}

# Grant CREATE SCHEMA system privilege
resource "exasol_system_privilege" "data_loader_create_schema" {
  grantee   = exasol_role.data_loader.name
  privilege = "CREATE SCHEMA"
}

# =============================================================================
# OBJECT PRIVILEGES - Using exasol_object_privilege
# =============================================================================

# Grant USAGE on schema to a role
resource "exasol_object_privilege" "analyst_schema_usage" {
  grantee     = exasol_role.analyst.name
  privileges  = ["USAGE"]
  object_type = "SCHEMA"
  object_name = exasol_schema.analytics.name
}

# Grant multiple privileges on schema
resource "exasol_object_privilege" "analyst_schema_read_write" {
  grantee     = exasol_role.analyst.name
  privileges  = ["USAGE", "SELECT", "INSERT", "UPDATE", "DELETE"]
  object_type = "SCHEMA"
  object_name = exasol_schema.analytics.name
}

# Grant ALL privileges on schema
resource "exasol_object_privilege" "data_loader_all_schema" {
  grantee     = exasol_role.data_loader.name
  privileges  = ["ALL"]
  object_type = "SCHEMA"
  object_name = exasol_schema.analytics.name
}

# Grant SELECT on a table
resource "exasol_object_privilege" "analyst_select_sales" {
  grantee     = exasol_role.analyst.name
  privileges  = ["SELECT"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# Grant INSERT on a table
resource "exasol_object_privilege" "data_loader_insert_sales" {
  grantee     = exasol_role.data_loader.name
  privileges  = ["INSERT"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# Grant multiple privileges on a table
resource "exasol_object_privilege" "data_loader_write_sales" {
  grantee     = exasol_role.data_loader.name
  privileges  = ["UPDATE", "DELETE"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# Grant ALL on a view
resource "exasol_object_privilege" "analyst_all_view" {
  grantee     = exasol_role.analyst.name
  privileges  = ["ALL"]
  object_type = "VIEW"
  object_name = "ANALYTICS.MONTHLY_SUMMARY"
}

# =============================================================================
# ROLE GRANTS - Using exasol_role_grant
# =============================================================================

# Grant role to user
resource "exasol_role_grant" "john_analyst_role" {
  role    = exasol_role.analyst.name
  grantee = exasol_user.john.name
}

# Grant role to user with admin option
resource "exasol_role_grant" "jane_analyst_role" {
  role              = exasol_role.analyst.name
  grantee           = exasol_user.jane.name
  with_admin_option = true
}

# Grant role to another role (role hierarchy)
resource "exasol_role_grant" "data_loader_to_analyst" {
  role    = exasol_role.data_loader.name
  grantee = exasol_role.analyst.name
}
