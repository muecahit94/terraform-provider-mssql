---
page_title: "mssql_database_object_permission Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a permission granted on an object of a database or on one of its columns.
---

# mssql_database_object_permission (Resource)

Manages a permission granted on an object of a database (a table, view, procedure or function) or on one column of it,
with `GRANT ... ON OBJECT::[schema].[object]`. Permissions on the whole database or on a schema are managed with
[`mssql_database_permission`](database_permission.md) and [`mssql_schema_permission`](schema_permission.md).

## Example Usage

```hcl
# Read a table
resource "mssql_database_object_permission" "orders_select" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "Orders"
  principal_name = mssql_database_role.readers.name
  permission     = "SELECT"
}

# Update one column only
resource "mssql_database_object_permission" "orders_total_update" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "Orders"
  column_name    = "Total"
  principal_name = mssql_sql_user.app.name
  permission     = "UPDATE"
}

# Run a stored procedure
resource "mssql_database_object_permission" "close_execute" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "usp_CloseOrders"
  principal_name = mssql_sql_user.app.name
  permission     = "EXECUTE"
}
```

## Argument Reference

All arguments force a new resource when they change.

- `database_name` - (Required) The name of the database.
- `schema_name` - (Required) The schema of the object.
- `object_name` - (Required) The name of the table, view, procedure or function.
- `column_name` - (Optional) The column, for a column-level permission such as `SELECT` or `UPDATE` on one column. Omit it for a permission on the whole object.
- `principal_name` - (Required) The user or role that gets the permission.
- `permission` - (Required) The permission, in upper case: for example `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `EXECUTE`, `REFERENCES`, `ALTER`, `CONTROL` or `VIEW DEFINITION`.
- `with_grant_option` - (Optional) Whether the principal can grant the permission to others (`WITH GRANT OPTION`). Defaults to `false`.

## Attribute Reference

- `id` - The ID: `database/schema/object[/column]/principal/permission`.

## Behaviour to know

- **A `DENY` is never removed silently.** `GRANT` over a `DENY` replaces it, so when a `DENY` exists for the same
  principal, permission and object, creating the grant fails with an explanation. A `DENY` made outside Terraform on a
  managed grant makes the grant disappear from the state, and the next apply then stops at that error until the `DENY`
  is removed.
- Deleting the resource runs `REVOKE ... CASCADE`, which also revokes what the principal has granted on to others.
- Objects of system schemas can be granted on as well (for example `SELECT` on `sys.all_columns`).

## Import

Permissions can be imported with `database/schema/object/principal/permission`, or
`database/schema/object/column/principal/permission` for a column-level permission:

```shell
terraform import mssql_database_object_permission.orders_select app_db/sales/Orders/readers/SELECT
terraform import mssql_database_object_permission.orders_total_update app_db/sales/Orders/Total/app_user/UPDATE
```

Names that contain a `/` cannot be imported.
