---
page_title: "mssql_databases Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Get information about all SQL Server databases.
---

# mssql_databases (Data Source)

Use this data source to list all databases on the SQL Server.

## Example Usage

```hcl
data "mssql_databases" "all" {}

output "database_names" {
  value = [for db in data.mssql_databases.all.databases : db.name]
}

# Databases that are not in the SIMPLE recovery model
output "databases_with_log_backups" {
  value = [for db in data.mssql_databases.all.databases : db.name if db.recovery_model != "SIMPLE"]
}
```

## Argument Reference

This data source has no required arguments.

## Attribute Reference

- `databases` - A list of databases, each with:
  - `id` - The database ID.
  - `name` - The database name.
  - `collation` - The collation of the database.
  - `compatibility_level` - The compatibility level of the database.
  - `recovery_model` - The recovery model: `FULL`, `SIMPLE` or `BULK_LOGGED`.
  - `owner_name` - The login that owns the database; empty when the owner login no longer exists.
