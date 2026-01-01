---
page_title: "mssql_azuread_user Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about an Azure AD user.
---

# mssql_azuread_user (Data Source)

Use this data source to get information about an Azure AD user in a database.

## Example Usage

```hcl
data "mssql_azuread_user" "example" {
  database_name = "mydb"
  name          = "user@example.com"
}

output "default_schema" {
  value = data.mssql_azuread_user.example.default_schema
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the Azure AD user.

## Attribute Reference

- `id` - The ID of the user in format `database_id/principal_id`.
- `default_schema` - The default schema of the user.
