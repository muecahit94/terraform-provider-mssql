data "mssql_database_permissions" "example" {
  database_name  = "example_db"
  principal_name = "example_user"
}
