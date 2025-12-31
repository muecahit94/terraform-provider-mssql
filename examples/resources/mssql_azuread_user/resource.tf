resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_azuread_user" "example" {
  name          = "example_ad_user"
  database_name = mssql_database.example.name
  object_id     = "00000000-0000-0000-0000-000000000000"
}
