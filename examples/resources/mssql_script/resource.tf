resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_script" "example" {
  database_name = mssql_database.example.name
  create_script = "CREATE TABLE example_table (id int)"
  delete_script = "DROP TABLE example_table"
}
