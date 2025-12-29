---
page_title: "mssql_sql_login Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server login.
---

# mssql_sql_login (Resource)

Manages a SQL Server login with password authentication.

## Example Usage

### Basic Login

```hcl
resource "mssql_sql_login" "example" {
  name     = "my_login"
  password = "SecurePassword123!"
}
```

### Login with All Options

```hcl
resource "mssql_sql_login" "full_example" {
  name                     = "app_login"
  password                 = "SecurePassword123!"
  default_database         = mssql_database.app.name
  check_expiration_enabled = true
  check_policy_enabled     = true
  is_disabled              = false
}
```

## Argument Reference

- `name` - (Required) The name of the login. Changing this forces a new resource.
- `password` - (Required) The password for the login.
- `default_database` - (Optional) The default database for the login. Defaults to `master`.
- `default_language` - (Optional) The default language for the login.
- `check_expiration_enabled` - (Optional) Whether password expiration is checked. Defaults to `false`.
- `check_policy_enabled` - (Optional) Whether password policy is enforced. Defaults to `true`.
- `is_disabled` - (Optional) Whether the login is disabled. Defaults to `false`.

## Attribute Reference

- `id` - The login principal ID.

## Import

Logins can be imported using the login name:

```shell
terraform import mssql_sql_login.example my_login
```
