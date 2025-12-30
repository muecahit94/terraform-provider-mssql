terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 1.0"
    }
  }
}

provider "mssql" {
  hostname = var.sql_hostname
  port     = var.sql_port

  sql_auth {
    username = var.sql_username
    password = var.sql_password
  }
}

# Create a database
resource "mssql_database" "app" {
  name = "application_db"
}

# Create a login for the application
resource "mssql_sql_login" "app" {
  name             = "app_login"
  password         = var.app_password
  default_database = mssql_database.app.name
}

# Create a user in the database
resource "mssql_sql_user" "app" {
  database_name  = mssql_database.app.name
  name           = "app_user"
  login_name     = mssql_sql_login.app.name
  default_schema = "app"
}

# Create a schema for the application
resource "mssql_schema" "app" {
  database_name = mssql_database.app.name
  name          = "app"
  owner_name    = mssql_sql_user.app.name
}

# Create a role for read-only access
resource "mssql_database_role" "readers" {
  database_name = mssql_database.app.name
  name          = "app_readers"
}

# Add the user to the role
resource "mssql_database_role_member" "app_reader" {
  database_name = mssql_database.app.name
  role_name     = mssql_database_role.readers.name
  member_name   = mssql_sql_user.app.name
}

# Grant SELECT permission to the role
resource "mssql_database_permission" "readers_select" {
  database_name  = mssql_database.app.name
  principal_name = mssql_database_role.readers.name
  permission     = "SELECT"
}

# Grant EXECUTE permission on the schema
resource "mssql_schema_permission" "app_execute" {
  database_name     = mssql_database.app.name
  schema_name       = mssql_schema.app.name
  principal_name    = mssql_sql_user.app.name
  permission        = "EXECUTE"
  with_grant_option = true
}