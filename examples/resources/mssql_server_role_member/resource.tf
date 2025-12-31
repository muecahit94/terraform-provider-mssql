resource "mssql_server_role" "example" {
  name = "example_server_role"
}

resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecretPassword123!"
}

resource "mssql_server_role_member" "example" {
  role_name   = mssql_server_role.example.name
  member_name = mssql_sql_login.example.name
}
