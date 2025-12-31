resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_database_role" "example" {
  name          = "example_role"
  database_name = mssql_database.example.name
}
