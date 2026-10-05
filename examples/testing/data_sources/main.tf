terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 1.0"
    }
  }
}

provider "mssql" {
  hostname = "localhost"
  port     = 1433

  sql_auth {
    username = "sa"
    password = "P@ssw0rd123!"
  }
}

# List all databases
data "mssql_databases" "all" {}

# Get specific database info
data "mssql_database" "master" {
  name = "master"
}

# Execute a custom query
data "mssql_query" "version" {
  query = "SELECT @@VERSION as version"
}

# List all logins
data "mssql_sql_logins" "all" {}

# Get server roles
data "mssql_server_roles" "all" {}

# Get specific login info using provider default
data "mssql_sql_login" "sa_default" {
  name = "sa"
}

# Get specific login info using server override block and login_name alias
data "mssql_sql_login" "sa_server_override" {
  server {
    host = "localhost"
    port = 1433
    login {
      username = "sa"
      password = "P@ssw0rd123!"
    }
  }

  login_name = "sa"
}

output "databases" {
  value = [for db in data.mssql_databases.all.databases : db.name]
}

output "sql_version" {
  value = data.mssql_query.version.result[0].values["version"]
}

output "login_count" {
  value = length(data.mssql_sql_logins.all.logins)
}

output "server_roles" {
  value = [for role in data.mssql_server_roles.all.roles : role.name]
}

output "sa_login_name" {
  value = data.mssql_sql_login.sa_server_override.name
}

output "sa_login_sid" {
  value = data.mssql_sql_login.sa_server_override.sid
}

