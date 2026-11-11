# Using default provider connection
data "mssql_sql_login" "example" {
  name = "example_login"
}

# Using named server connection from provider servers map
data "mssql_sql_login" "registered" {
  server_key = "primary"
  name       = "app_user"
}

# Using per-data-source server override and login_name alias
data "mssql_sql_login" "cluster_user" {
  server {
    hostname = "sql-cluster.internal"
    port     = 1433
    sql_auth {
      username = "sa"
      password = "SecretPassword123!"
    }
  }

  login_name = "app_user"
}

