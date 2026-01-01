---
page_title: "mssql_database_role Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about a database role.
---

# mssql_database_role (Data Source)

Use this data source to get information about a database role.

## Example Usage

```hcl
data "mssql_database_role" "example" {
  database_name = "mydb"
  name          = "app_readers"
}

output "owner" {
  value = data.mssql_database_role.example.owner_name
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the role.

## Attribute Reference

- `id` - The ID of the role in format `database_id/principal_id`.
- `owner_name` - The name of the role owner.
