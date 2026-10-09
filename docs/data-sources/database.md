---
page_title: "mssql_database Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Get information about a SQL Server database.
---

# mssql_database (Data Source)

Use this data source to get information about a SQL Server database.

## Example Usage

```hcl
data "mssql_database" "master" {
  name = "master"
}

output "database_id" {
  value = data.mssql_database.master.id
}
```

## Argument Reference

- `name` - (Required) The name of the database.

## Attribute Reference

- `id` - The database ID.
- `collation` - The collation of the database.
- `compatibility_level` - The compatibility level of the database, for example `160`.
- `recovery_model` - The recovery model: `FULL`, `SIMPLE` or `BULK_LOGGED`.
