---
page_title: "mssql_schema Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server database schema.
---

# mssql_schema (Resource)

Manages a database schema.

## Example Usage

```hcl
resource "mssql_schema" "example" {
  database_name = mssql_database.example.name
  name          = "app"
  owner_name    = mssql_sql_user.admin.name
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the schema.
- `owner_name` - (Optional) The owner of the schema.

## Import

```shell
terraform import mssql_schema.example my_database/app
```
