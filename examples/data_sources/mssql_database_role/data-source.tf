data "mssql_database_role" "example" {
  name          = "example_role"
  database_name = "example_db"
}
