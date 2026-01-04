resource "mssql_database" "example" {
  name = "example_db"
}

# Email-based user with roles (object_id not required)
resource "mssql_azuread_user" "email_user" {
  name          = "john.doe@contoso.com"
  database_name = mssql_database.example.name
  roles         = ["db_datareader"]
}

# Managed identity with roles (object_id required)
resource "mssql_azuread_user" "managed_identity" {
  name          = "my-managed-identity"
  database_name = mssql_database.example.name
  object_id     = "00000000-0000-0000-0000-000000000000"
  roles         = ["db_datareader", "db_datawriter"]
}
