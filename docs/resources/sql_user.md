---
page_title: "mssql_sql_user Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server database user mapped to a login.
---

# mssql_sql_user (Resource)

Manages a database user that is mapped to a SQL Server login.

## Example Usage

```hcl
resource "mssql_database" "example" {
  name = "my_database"
}

resource "mssql_sql_login" "example" {
  name     = "my_login"
  password = "SecurePassword123!"
}

resource "mssql_sql_user" "example" {
  database_name  = mssql_database.example.name
  name           = "my_user"
  login_name     = mssql_sql_login.example.name
  default_schema = "dbo"
}
```

## Argument Reference

- `database_name` - (Required) The name of the database. Changing this forces a new resource.
- `name` - (Required) The name of the user. Changing this forces a new resource.
- `login_name` - (Required) The name of the login to map this user to. Changing this forces a new resource.
- `default_schema` - (Optional) The default schema for the user. Defaults to `dbo`.

## Attribute Reference

- `id` - The user ID in format `database_id/principal_id`.
- `default_schema` - The default schema for the user.

## Import

Users can be imported using `database_name/user_name`:

```shell
terraform import mssql_sql_user.example my_database/my_user
```
