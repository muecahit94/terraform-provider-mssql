terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 0.0"
    }
  }
}

provider "mssql" {
  hostname = var.sql_hostname
  port     = 1433

  azure_auth {
    client_id     = var.azure_client_id
    client_secret = var.azure_client_secret
    tenant_id     = var.azure_tenant_id
  }
}

# Create an Azure AD user
resource "mssql_azuread_user" "developer" {
  database_name  = var.database_name
  name           = var.developer_email
  object_id      = var.developer_object_id
  default_schema = "dbo"
}

# Create an Azure AD service principal for the application
resource "mssql_azuread_service_principal" "app" {
  database_name  = var.database_name
  name           = var.app_name
  client_id      = var.app_client_id
  default_schema = "dbo"
}

# Grant SELECT permission to the application
resource "mssql_database_permission" "app_select" {
  database_name  = var.database_name
  principal_name = mssql_azuread_service_principal.app.name
  permission     = "SELECT"
}

# Grant EXECUTE permission to the application
resource "mssql_database_permission" "app_execute" {
  database_name  = var.database_name
  principal_name = mssql_azuread_service_principal.app.name
  permission     = "EXECUTE"
}
