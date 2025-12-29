---
page_title: "mssql_azuread_user Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages an Azure AD user in a SQL Server database.
---

# mssql_azuread_user (Resource)

Manages an Azure AD user in a database. Requires Azure AD authentication.

## Example Usage

```hcl
resource "mssql_azuread_user" "example" {
  database_name  = mssql_database.example.name
  name           = "john.doe@contoso.com"
  object_id      = "00000000-0000-0000-0000-000000000000"
  default_schema = "dbo"
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The display name of the Azure AD user.
- `object_id` - (Required) The Azure AD object ID of the user.
- `default_schema` - (Optional) The default schema for the user. Defaults to `dbo`.

## Import

```shell
terraform import mssql_azuread_user.example my_database/john.doe@contoso.com
```
