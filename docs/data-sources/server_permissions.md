---
page_title: "mssql_server_permissions Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get server permissions for a principal.
---

# mssql_server_permissions (Data Source)

Use this data source to get all server-level permissions granted to a specific principal.

## Example Usage

```hcl
data "mssql_server_permissions" "example" {
  principal_name = "app_login"
}

output "permissions" {
  value = data.mssql_server_permissions.example.permissions
}
```

## Argument Reference

- `principal_name` - (Required) The name of the principal (login) to get permissions for.

## Attribute Reference

- `permissions` - A list of permissions. Each permission contains:
  - `permission` - The permission name (e.g., VIEW SERVER STATE, CONTROL SERVER).
  - `with_grant_option` - Whether the permission was granted with GRANT OPTION.
