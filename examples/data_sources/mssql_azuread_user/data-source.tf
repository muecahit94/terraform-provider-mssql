data "mssql_azuread_user" "example" {
  name          = "example_ad_user"
  database_name = "example_db"
}
