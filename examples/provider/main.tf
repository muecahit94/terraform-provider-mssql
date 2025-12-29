terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 0.1"
    }
  }
}

# SQL Authentication
provider "mssql" {
  hostname = "localhost"
  port     = 1433

  sql_auth = {
    username = "sa"
    password = "P@ssw0rd123!"
  }
}

# Create a database
resource "mssql_database" "example" {
  name = "example_db"
}

# Create a login
resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecurePassword123!"
}

# Create a user in the database
resource "mssql_sql_user" "example" {
  database_name = mssql_database.example.name
  name          = "example_user"
  login_name    = mssql_sql_login.example.name
}

# Grant permissions
resource "mssql_database_permission" "example" {
  database_name  = mssql_database.example.name
  principal_name = mssql_sql_user.example.name
  permission     = "SELECT"
}
