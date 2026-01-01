---
page_title: "mssql_schema Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about a database schema.
---

# mssql_schema (Data Source)

Use this data source to get information about a database schema.

## Example Usage

```hcl
data "mssql_schema" "example" {
  database_name = "mydb"
  name          = "app"
}

output "owner" {
  value = data.mssql_schema.example.owner_name
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the schema.

## Attribute Reference

- `id` - The ID of the schema in format `database_id/schema_id`.
- `owner_name` - The name of the schema owner.
