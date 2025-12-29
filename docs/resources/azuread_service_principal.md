---
page_title: "mssql_azuread_service_principal Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages an Azure AD service principal in a SQL Server database.
---

# mssql_azuread_service_principal (Resource)

Manages an Azure AD service principal (application) in a database.

## Example Usage

```hcl
resource "mssql_azuread_service_principal" "example" {
  database_name  = mssql_database.example.name
  name           = "my-application"
  client_id      = "00000000-0000-0000-0000-000000000000"
  default_schema = "dbo"
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The display name of the service principal.
- `client_id` - (Required) The Azure AD client (application) ID.
- `default_schema` - (Optional) The default schema. Defaults to `dbo`.

## Import

```shell
terraform import mssql_azuread_service_principal.example my_database/my-application
```
