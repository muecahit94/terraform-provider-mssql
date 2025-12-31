resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecretPassword123!"
}

resource "mssql_sql_user" "example" {
  name          = "example_user"
  database_name = mssql_database.example.name
  login_name    = mssql_sql_login.example.name
}
