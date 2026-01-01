---
page_title: "mssql_schema_permissions Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get schema permissions for a principal.
---

# mssql_schema_permissions (Data Source)

Use this data source to get all schema permissions granted to a specific principal.

## Example Usage

```hcl
data "mssql_schema_permissions" "example" {
  database_name  = "mydb"
  schema_name    = "app"
  principal_name = "app_user"
}

output "permissions" {
  value = data.mssql_schema_permissions.example.permissions
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `schema_name` - (Required) The name of the schema.
- `principal_name` - (Required) The name of the principal to get permissions for.

## Attribute Reference

- `permissions` - A list of permissions. Each permission contains:
  - `permission` - The permission name (e.g., SELECT, INSERT, EXECUTE).
  - `with_grant_option` - Whether the permission was granted with GRANT OPTION.
