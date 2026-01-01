---
page_title: "mssql_server_permission Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a server-level permission grant.
---

# mssql_server_permission (Resource)

Grants a server-level permission for a principal.

## Example Usage

```hcl
resource "mssql_server_permission" "example" {
  principal_name = mssql_sql_login.app.name
  permission     = "VIEW SERVER STATE"
}
```

## Argument Reference

- `principal_name` - (Required) The name of the login.
- `permission` - (Required) The permission to grant.
- `with_grant_option` - (Optional) Whether the principal can grant this permission to others.

## Attribute Reference

- `id` - The permission ID in format `principal_name/permission`.

## Import

```shell
terraform import mssql_server_permission.example my_login/VIEW SERVER STATE
```
