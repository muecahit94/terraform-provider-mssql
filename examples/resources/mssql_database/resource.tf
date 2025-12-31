resource "mssql_database" "example" {
  name      = "example_db"
  collation = "SQL_Latin1_General_CP1_CI_AS"
}
