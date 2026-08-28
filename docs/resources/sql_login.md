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

### Login with Custom SID

```hcl
resource "mssql_sql_login" "with_sid" {
  name     = "mirrored_login"
  password = "SecurePassword123!"
  sid      = "0x0123456789ABCDEF0123456789ABCDEF"
}
```

### Login with a Write-Only Password

`password_wo` is a [write-only attribute](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only): it accepts
[ephemeral](https://developer.hashicorp.com/terraform/language/resources/ephemeral) values and Terraform writes it to neither the plan nor
the state file. It requires Terraform 1.11 or later.

```hcl
ephemeral "random_password" "login" {
  length           = 32
  override_special = "!#$*()-_+[]{}<>?"
}

resource "mssql_sql_login" "example" {
  name                = "my_login"
  password_wo         = ephemeral.random_password.login.result
  password_wo_version = "1"
}
```

Because the value is not stored, Terraform cannot detect that it changed. The
provider issues an `ALTER LOGIN` only when `password_wo_version` changes, so bump
it in the same apply that rotates the password:

```hcl
ephemeral "random_password" "login" {
  length           = 32
  override_special = "!#$*()-_+[]{}<>?"
  keepers          = { rotation = "2" }
}

resource "mssql_sql_login" "example" {
  name                = "my_login"
  password_wo         = ephemeral.random_password.login.result
  password_wo_version = "2"
}
```

A write-only password never reaches the state, so nothing can read it back
afterwards — write it to a secret store in the same apply (for example an
`aws_secretsmanager_secret_version` with `secret_string_wo`), or consumers will
have no way to obtain it.

Moving an existing login from `password` to `password_wo` needs no version bump:
the old value is still in state, so the provider applies the write-only password
once and drops the old value from state.

### Login with All Options

```hcl
resource "mssql_sql_login" "full_example" {
  name                     = "app_login"
  password                 = "SecurePassword123!"
  sid                      = "0x0123456789ABCDEF0123456789ABCDEF"
  default_database         = mssql_database.app.name
  check_expiration_enabled = true
  check_policy_enabled     = true
  is_disabled              = false
}
```

## Argument Reference

- `name` - (Required) The name of the login. Changing this forces a new resource.
- `password` - (Optional) The password for the login. Persisted in the plan and state files. Exactly one of `password` and `password_wo` must be set.
- `password_wo` - (Optional, [write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only)) The password for the login. Accepts ephemeral values and is written to neither the plan nor the state file. Requires Terraform 1.11 or later. Exactly one of `password` and `password_wo` must be set.
- `password_wo_version` - (Optional) An arbitrary token whose change triggers an `ALTER LOGIN` with the current `password_wo` value. Only valid together with `password_wo`. Without it, a rotated `password_wo` is never applied.
- `sid` - (Optional) The SID (Security Identifier) of the login in hexadecimal format (e.g., `0x0123456789ABCDEF0123456789ABCDEF`). Changing this forces a new resource. If not specified, SQL Server generates a SID automatically.
- `default_database` - (Optional) The default database for the login. Defaults to `master`.
- `default_language` - (Optional) The default language for the login.
- `check_expiration_enabled` - (Optional) Whether password expiration is checked. Defaults to `false`.
- `check_policy_enabled` - (Optional) Whether password policy is enforced. Defaults to `true`.
- `is_disabled` - (Optional) Whether the login is disabled. Defaults to `false`.

## Attribute Reference

- `id` - The login principal ID.
- `sid` - The SID (Security Identifier) of the SQL login in hexadecimal format.

## Import

Logins can be imported using the login name:

```shell
terraform import mssql_sql_login.example my_login
```
