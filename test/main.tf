terraform {
  required_providers {
    exasol = {
      source = "local/exasol/exasol"
      version = "0.1.1"
    }
  }
}

provider "exasol" {
  host                       = "localhost"
  port                       = 8563
  user                       = "sys"
  password                   = "exasol"
  validate_server_certificate = false
}

# Test role
resource "exasol_role" "test_role" {
  name = "TEST_OWNER_ROLE"
}

# Test schema with owner attribute (NEW FEATURE)
resource "exasol_schema" "test_schema" {
  name  = "TEST_SCHEMA_WITH_OWNER"
  owner = exasol_role.test_role.name
}

# Test connection
resource "exasol_connection" "test_s3" {
  name     = "TEST_S3_CONNECTION"
  to       = "https://test-bucket.s3.us-east-1.amazonaws.com"
  user     = "test_access_key"
  password = "test_secret_key"
}

# Test connection grant (NEW RESOURCE)
resource "exasol_connection_grant" "test_grant" {
  connection_name = exasol_connection.test_s3.name
  grantee         = exasol_role.test_role.name
}
