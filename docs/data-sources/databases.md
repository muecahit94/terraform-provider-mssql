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
```

## Argument Reference

This data source has no required arguments.

## Attribute Reference

- `databases` - A list of databases, each with:
  - `id` - The database ID.
  - `name` - The database name.
