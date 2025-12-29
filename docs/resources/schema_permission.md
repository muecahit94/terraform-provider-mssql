---
page_title: "mssql_schema_permission Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a schema-level permission grant.
---

# mssql_schema_permission (Resource)

Grants a schema-level permission for a principal.

## Example Usage

```hcl
resource "mssql_schema_permission" "example" {
  database_name  = mssql_database.example.name
  schema_name    = mssql_schema.app.name
  principal_name = mssql_sql_user.app.name
  permission     = "SELECT"
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `schema_name` - (Required) The name of the schema.
- `principal_name` - (Required) The name of the principal.
- `permission` - (Required) The permission to grant.
- `with_grant_option` - (Optional) Whether the principal can grant this permission to others.

## Import

```shell
terraform import mssql_schema_permission.example my_database/app/my_user/SELECT
```
