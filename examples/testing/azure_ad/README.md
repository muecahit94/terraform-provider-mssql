# Azure AD Authentication Example

This example demonstrates using Azure AD authentication with the provider.

```hcl
terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 0.0"
    }
  }
}

# Using Service Principal
provider "mssql" {
  hostname = "myserver.database.windows.net"
  port     = 1433

  azure_auth {
    client_id     = var.azure_client_id
    client_secret = var.azure_client_secret
    tenant_id     = var.azure_tenant_id
  }
}

# Create an Azure AD user
resource "mssql_azuread_user" "example" {
  database_name  = "my_database"
  name           = "john.doe@contoso.com"
  object_id      = "00000000-0000-0000-0000-000000000000"
  default_schema = "dbo"
}

# Create an Azure AD service principal
resource "mssql_azuread_service_principal" "app" {
  database_name  = "my_database"
  name           = "my-application"
  client_id      = "00000000-0000-0000-0000-000000000000"
  default_schema = "dbo"
}

# Grant permissions to the service principal
resource "mssql_database_permission" "app_select" {
  database_name  = "my_database"
  principal_name = mssql_azuread_service_principal.app.name
  permission     = "SELECT"
}
```
