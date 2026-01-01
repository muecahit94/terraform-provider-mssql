---
page_title: "mssql_sql_users Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about all SQL users in a database.
---

# mssql_sql_users (Data Source)

Use this data source to list all SQL users in a database.

## Example Usage

```hcl
data "mssql_sql_users" "example" {
  database_name = "mydb"
}

output "user_names" {
  value = [for u in data.mssql_sql_users.example.users : u.name]
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.

## Attribute Reference

- `users` - A list of users. Each user contains:
  - `id` - The ID of the user in format `database_id/principal_id`.
  - `database_name` - The database name.
  - `name` - The name of the user.
  - `login_name` - The login name associated with the user.
  - `default_schema` - The default schema of the user.
