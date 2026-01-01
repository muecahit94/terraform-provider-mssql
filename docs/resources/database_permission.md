---
page_title: "mssql_database_permission Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a database-level permission grant.
---

# mssql_database_permission (Resource)

Grants or manages a database-level permission for a principal.

## Example Usage

```hcl
resource "mssql_database_permission" "select" {
  database_name  = mssql_database.example.name
  principal_name = mssql_sql_user.app.name
  permission     = "SELECT"
}

resource "mssql_database_permission" "with_grant" {
  database_name     = mssql_database.example.name
  principal_name    = mssql_sql_user.admin.name
  permission        = "CONTROL"
  with_grant_option = true
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `principal_name` - (Required) The name of the principal (user or role).
- `permission` - (Required) The permission to grant (e.g., SELECT, INSERT, UPDATE, DELETE, EXECUTE, CONTROL).
- `with_grant_option` - (Optional) Whether the principal can grant this permission to others. Defaults to `false`.

## Attribute Reference

- `id` - The permission ID in format `database_name/principal_name/permission`.

## Import

```shell
terraform import mssql_database_permission.example my_database/my_user/SELECT
```
