---
page_title: "mssql_azuread_service_principal Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about an Azure AD service principal.
---

# mssql_azuread_service_principal (Data Source)

Use this data source to get information about an Azure AD service principal in a database.

## Example Usage

```hcl
data "mssql_azuread_service_principal" "example" {
  database_name = "mydb"
  name          = "my-app-sp"
}

output "default_schema" {
  value = data.mssql_azuread_service_principal.example.default_schema
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the Azure AD service principal.

## Attribute Reference

- `id` - The ID of the service principal in format `database_id/principal_id`.
- `default_schema` - The default schema of the service principal.
