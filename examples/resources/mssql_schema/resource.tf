resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_sql_login" "owner" {
  name     = "example_owner_login"
  password = "SecretPassword123!"
}

resource "mssql_sql_user" "owner" {
  name          = "example_owner_user"
  database_name = mssql_database.example.name
  login_name    = mssql_sql_login.owner.name
}

resource "mssql_schema" "example" {
  name          = "example_schema"
  database_name = mssql_database.example.name
  owner_name    = mssql_sql_user.owner.name
}
