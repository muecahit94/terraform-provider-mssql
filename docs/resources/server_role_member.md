---
page_title: "mssql_server_role_member Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages membership in a SQL Server server role.
---

# mssql_server_role_member (Resource)

Manages membership of a login in a server role.

## Example Usage

```hcl
resource "mssql_server_role_member" "example" {
  role_name   = "sysadmin"
  member_name = mssql_sql_login.admin.name
}
```

## Argument Reference

- `role_name` - (Required) The name of the server role.
- `member_name` - (Required) The name of the login.

## Import

```shell
terraform import mssql_server_role_member.example sysadmin/my_login
```
