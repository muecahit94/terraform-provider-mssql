---
page_title: "mssql_server_role Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server server role.
---

# mssql_server_role (Resource)

Manages a custom server-level role.

## Example Usage

```hcl
resource "mssql_server_role" "example" {
  name = "app_admins"
}
```

## Argument Reference

- `name` - (Required) The name of the role.
- `owner_name` - (Optional) The owner of the role.

## Import

```shell
terraform import mssql_server_role.example app_admins
```
