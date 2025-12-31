resource "mssql_database" "example" {
  name = "example_db"
}

resource "mssql_azuread_service_principal" "example" {
  name          = "example_ad_sp"
  database_name = mssql_database.example.name
  client_id     = "00000000-0000-0000-0000-000000000000"
}
