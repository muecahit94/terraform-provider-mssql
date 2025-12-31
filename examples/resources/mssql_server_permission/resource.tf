resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecretPassword123!"
}

resource "mssql_server_permission" "example" {
  principal_name = mssql_sql_login.example.name
  permission     = "CONNECT SQL"
}
