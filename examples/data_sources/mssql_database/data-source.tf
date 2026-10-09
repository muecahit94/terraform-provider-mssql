data "mssql_database" "example" {
  name = "example_db"
}

output "example_collation" {
  value = data.mssql_database.example.collation
}
