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

# Grant role to user
resource "exasol_role_grant" "analyst_to_user" {
  role    = exasol_role.analyst.name
  grantee = exasol_user.analyst_user.name
}

# Grant role to user with admin option (allows user to grant role to others)
resource "exasol_role_grant" "analyst_to_user_admin" {
  role              = exasol_role.analyst.name
  grantee           = "ANOTHER_USER"
  with_admin_option = true
}

# Grant role to another role (role hierarchy)
resource "exasol_role" "senior_analyst" {
  name = "SENIOR_ANALYST_ROLE"
}

resource "exasol_role_grant" "analyst_to_senior" {
  role    = exasol_role.analyst.name
  grantee = exasol_role.senior_analyst.name
}

# =============================================================================
# SYSTEM PRIVILEGES - System-level privileges
# =============================================================================

# Grant system privilege to a role
resource "exasol_system_privilege" "create_session" {
  grantee   = exasol_role.analyst.name
  privilege = "CREATE SESSION"
}

# Grant system privilege with admin option
resource "exasol_system_privilege" "use_any_schema" {
  grantee           = exasol_role.analyst.name
  privilege         = "USE ANY SCHEMA"
  with_admin_option = true
}

# Grant CREATE TABLE system privilege
resource "exasol_system_privilege" "create_table" {
  grantee   = exasol_role.analyst.name
  privilege = "CREATE TABLE"
}

# Grant IMPORT/EXPORT privileges for ETL
resource "exasol_system_privilege" "import_priv" {
  grantee   = exasol_role.analyst.name
  privilege = "IMPORT"
}

resource "exasol_system_privilege" "export_priv" {
  grantee   = exasol_role.analyst.name
  privilege = "EXPORT"
}

# =============================================================================
# OBJECT PRIVILEGES - Schema-level privileges
# =============================================================================

# Grant USAGE on a schema to a role
resource "exasol_object_privilege" "schema_usage" {
  grantee     = exasol_role.analyst.name
  privileges  = ["USAGE"]
  object_type = "SCHEMA"
  object_name = exasol_schema.example.name
}

# Grant multiple privileges on a schema
resource "exasol_object_privilege" "schema_read_write" {
  grantee     = exasol_role.analyst.name
  privileges  = ["USAGE", "SELECT", "INSERT", "UPDATE", "DELETE"]
  object_type = "SCHEMA"
  object_name = exasol_schema.example.name
}

# Grant ALL privileges on a schema
resource "exasol_object_privilege" "schema_all" {
  grantee     = "DBA_ROLE"
  privileges  = ["ALL"]
  object_type = "SCHEMA"
  object_name = exasol_schema.example.name
}

# =============================================================================
# OBJECT PRIVILEGES - Table-level privileges
# =============================================================================

# Grant SELECT on a table
resource "exasol_object_privilege" "table_select" {
  grantee     = exasol_role.analyst.name
  privileges  = ["SELECT"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# Grant multiple privileges on a table
resource "exasol_object_privilege" "table_write" {
  grantee     = "DATA_LOADER_ROLE"
  privileges  = ["INSERT", "UPDATE", "DELETE"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# Grant ALL on a table
resource "exasol_object_privilege" "table_all" {
  grantee     = "TABLE_OWNER"
  privileges  = ["ALL"]
  object_type = "TABLE"
  object_name = "ANALYTICS.SALES_DATA"
}

# =============================================================================
# OBJECT PRIVILEGES - View-level privileges
# =============================================================================

# Grant SELECT on a view
resource "exasol_object_privilege" "view_select" {
  grantee     = exasol_role.analyst.name
  privileges  = ["SELECT"]
  object_type = "VIEW"
  object_name = "ANALYTICS.SALES_SUMMARY"
}
