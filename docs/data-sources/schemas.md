---
page_title: "mssql_schemas Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about all schemas in a database.
---

# mssql_schemas (Data Source)

Use this data source to list all schemas in a database.

## Example Usage

```hcl
data "mssql_schemas" "all" {
  database_name = "mydb"
}

output "schema_names" {
  value = [for s in data.mssql_schemas.all.schemas : s.name]
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.

## Attribute Reference

- `schemas` - A list of schemas. Each schema contains:
  - `id` - The ID of the schema in format `database_id/schema_id`.
  - `database_name` - The database name.
  - `name` - The name of the schema.
  - `owner_name` - The name of the schema owner.
