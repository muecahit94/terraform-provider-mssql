resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecretPassword123!"
}
