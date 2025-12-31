data "mssql_azuread_service_principal" "example" {
  name          = "example_ad_sp"
  database_name = "example_db"
}
