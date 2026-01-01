---
page_title: "mssql_server_roles Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about all server roles.
---

# mssql_server_roles (Data Source)

Use this data source to list all server roles.

## Example Usage

```hcl
data "mssql_server_roles" "all" {}

output "role_names" {
  value = [for r in data.mssql_server_roles.all.roles : r.name]
}
```

## Argument Reference

This data source has no required arguments.

## Attribute Reference

- `roles` - A list of server roles. Each role contains:
  - `id` - The principal ID of the role.
  - `name` - The name of the role.
  - `owner_name` - The name of the role owner.
