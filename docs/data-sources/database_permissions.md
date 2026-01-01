---
page_title: "mssql_database_permissions Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get database permissions for a principal.
---

# mssql_database_permissions (Data Source)

Use this data source to get all database permissions granted to a specific principal.

## Example Usage

```hcl
data "mssql_database_permissions" "example" {
  database_name  = "mydb"
  principal_name = "app_role"
}

output "permissions" {
  value = data.mssql_database_permissions.example.permissions
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `principal_name` - (Required) The name of the principal (user, role) to get permissions for.

## Attribute Reference

- `permissions` - A list of permissions. Each permission contains:
  - `permission` - The permission name (e.g., SELECT, INSERT, EXECUTE).
  - `with_grant_option` - Whether the permission was granted with GRANT OPTION.
