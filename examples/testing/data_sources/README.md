# Data Sources Example

This example demonstrates using data sources to query existing resources.

```hcl
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
    password = var.sa_password
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
  query = "SELECT @@VERSION as version, GETDATE() as current_time"
}

# List all logins
data "mssql_sql_logins" "all" {}

# Get server roles
data "mssql_server_roles" "all" {}

output "databases" {
  value = [for db in data.mssql_databases.all.databases : db.name]
}

output "sql_version" {
  value = data.mssql_query.version.result[0].values["version"]
}

output "login_count" {
  value = length(data.mssql_sql_logins.all.logins)
}
```
