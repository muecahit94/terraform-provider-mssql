---
page_title: "mssql_database_role_member Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages membership in a SQL Server database role.
---

# mssql_database_role_member (Resource)

Manages membership of a user or role in a database role.

## Example Usage

```hcl
resource "mssql_database_role_member" "example" {
  database_name = mssql_database.example.name
  role_name     = mssql_database_role.readers.name
  member_name   = mssql_sql_user.app.name
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `role_name` - (Required) The name of the role.
- `member_name` - (Required) The name of the member (user or role).

## Import

```shell
terraform import mssql_database_role_member.example my_database/app_readers/my_user
```
