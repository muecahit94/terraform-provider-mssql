data "mssql_schema_permissions" "example" {
  database_name  = "example_db"
  schema_name    = "dbo"
  principal_name = "example_user"
}
