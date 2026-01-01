---
page_title: "mssql_database_roles Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about all roles in a database.
---

# mssql_database_roles (Data Source)

Use this data source to list all roles in a database.

## Example Usage

```hcl
data "mssql_database_roles" "example" {
  database_name = "mydb"
}

output "role_names" {
  value = [for r in data.mssql_database_roles.example.roles : r.name]
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.

## Attribute Reference

- `roles` - A list of roles. Each role contains:
  - `id` - The ID of the role in format `database_id/principal_id`.
  - `database_name` - The database name.
  - `name` - The name of the role.
  - `owner_name` - The name of the role owner.
