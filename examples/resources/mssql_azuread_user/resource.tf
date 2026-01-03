resource "mssql_database" "example" {
  name = "example_db"
}

# Email-based user (object_id not required)
resource "mssql_azuread_user" "email_user" {
  name          = "john.doe@contoso.com"
  database_name = mssql_database.example.name
}

# Managed identity (object_id required)
resource "mssql_azuread_user" "managed_identity" {
  name          = "my-managed-identity"
  database_name = mssql_database.example.name
  object_id     = "00000000-0000-0000-0000-000000000000"
}
