terraform {
  required_providers {
    exasol = {
      source  = "local/exasol/bi-terraform-provider-exasol"
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

# Example 1: Exasol-to-Exasol connection
resource "exasol_connection" "exa_remote" {
  name     = "REMOTE_EXASOL"
  to       = "192.168.1.10:8563"
  user     = "remote_user"
  password = "remote_password"
}

# Example 2: S3 connection for data import/export
resource "exasol_connection" "s3_bucket" {
  name     = "MY_S3_BUCKET"
  to       = "https://my-bucket.s3.us-east-1.amazonaws.com"
  user     = "AKIAIOSFODNN7EXAMPLE"
  password = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}

# Example 3: FTP connection
resource "exasol_connection" "ftp_server" {
  name     = "FTP_DATA_SOURCE"
  to       = "ftp://ftp.example.com:21/data"
  user     = "ftp_user"
  password = "ftp_password"
}

# Example 4: Oracle connection (with TNS connect string)
resource "exasol_connection" "oracle_db" {
  name = "ORACLE_PROD"
  to   = "(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=oracle.example.com)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=ORCL)))"
  user = "oracle_user"
  password = "oracle_password"
}

# Example 5: JDBC connection (MySQL)
resource "exasol_connection" "mysql_db" {
  name     = "MYSQL_SOURCE"
  to       = "jdbc:mysql://mysql.example.com:3306/mydb"
  user     = "mysql_user"
  password = "mysql_password"
}

# Example 6: Connection without credentials (for public resources)
resource "exasol_connection" "public_http" {
  name = "PUBLIC_DATA"
  to   = "https://data.example.com/public/"
}