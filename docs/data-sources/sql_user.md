---
page_title: "mssql_sql_user Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about a SQL Server database user.
---

# mssql_sql_user (Data Source)

Use this data source to get information about a SQL Server database user.

## Example Usage

```hcl
data "mssql_sql_user" "example" {
  database_name = "mydb"
  name          = "app_user"
}

output "login_name" {
  value = data.mssql_sql_user.example.login_name
}

output "default_schema" {
  value = data.mssql_sql_user.example.default_schema
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The name of the user.

## Attribute Reference

- `id` - The ID of the user in format `database_id/principal_id`.
- `login_name` - The login name associated with the user.
- `default_schema` - The default schema of the user.
