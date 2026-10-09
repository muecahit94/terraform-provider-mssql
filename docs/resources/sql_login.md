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

### Multi-Host Logins (Per-Resource Server Override)

When synchronizing logins with identical SIDs across multiple SQL Server instances (such as AlwaysOn Availability Group replicas or a fleet of database hosts), you can override the target server directly on the resource:

```hcl
locals {
  servers = {
    "primary"   = { host = "sql-01.corp.internal", sa_user = "sa", sa_pass = var.sa_password }
    "secondary" = { host = "sql-02.corp.internal", sa_user = "sa", sa_pass = var.sa_password }
  }
}

resource "mssql_sql_login" "cluster_user" {
  for_each = local.servers

  server {
    hostname = each.value.host
    port     = 1433
    sql_auth {
      username = each.value.sa_user
      password = each.value.sa_pass
    }
  }

  name     = "app_user"
  password = "SecurePassword123!"
  sid      = "0xFEEDFACE1234567890ABCDEF12345678"
}
```

### Existing Login Without a Password in the Configuration

The password of a login is not readable, so a login that already exists (imported, or created earlier) does not need
`password` or `password_wo` in the configuration. The provider then leaves its password alone: a plan after the import
shows no change, and an apply never resets the password that applications use.

```hcl
resource "mssql_sql_login" "existing" {
  name             = "n_dagster"
  default_database = "master"
}
```

A password is only required to **create** a login; without one the plan fails with *Missing password*. To start managing
the password later, add `password_wo` (and bump `password_wo_version` to rotate it).

## Argument Reference

- `name` - (Optional) The name of the login. Exactly one of `name` or `login_name` must be set. Changing this forces a new resource.
- `login_name` - (Optional) Alias for `name`. The name of the login. Exactly one of `name` or `login_name` must be set. Changing this forces a new resource.
- `password` - (Optional) The password for the login. Persisted in the plan and state files. At most one of `password` and `password_wo` can be set; one of them is needed to create the login.
- `password_wo` - (Optional, [write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only)) The password for the login. Accepts ephemeral values and is written to neither the plan nor the state file. Requires Terraform 1.11 or later. At most one of `password` and `password_wo` can be set; one of them is needed to create the login.
- `password_wo_version` - (Optional) An arbitrary token whose change triggers an `ALTER LOGIN` with the current `password_wo` value. Only valid together with `password_wo`. Without it, a rotated `password_wo` is never applied.
- `sid` - (Optional) The SID (Security Identifier) of the login in hexadecimal format (e.g., `0x0123456789ABCDEF0123456789ABCDEF`). Changing this forces a new resource. If not specified, SQL Server generates a SID automatically.
- `default_database` - (Optional) The default database for the login. Defaults to `master`.
- `default_language` - (Optional) The default language for the login.
- `check_expiration_enabled` - (Optional) Whether password expiration is checked. Defaults to `false`.
- `check_policy_enabled` - (Optional) Whether password policy is enforced. Defaults to `true`.
- `is_disabled` - (Optional) Whether the login is disabled. Defaults to `false`.
- `server` - (Optional) SQL Server instance configuration block. When omitted, the resource uses the default provider-level connection.
  - `hostname` - (Optional) FQDN or IP address of the target SQL endpoint. Changing this forces a new resource.
  - `host` - (Optional) Alias for `hostname`.
  - `port` - (Optional) TCP port of SQL endpoint. Defaults to `1433`. Changing this forces a new resource.
  - `sql_auth` - (Optional) Block for SQL authentication credentials:
    - `username` - (Optional) Username for SQL authentication.
    - `password` - (Optional, Sensitive) Password for SQL authentication.
  - `login` - (Optional) Alias for `sql_auth`.
  - `azure_auth` - (Optional) Block for Azure AD authentication:
    - `client_id` - (Optional) Service Principal client ID.
    - `client_secret` - (Optional, Sensitive) Service Principal secret.
    - `tenant_id` - (Optional) Azure AD tenant ID.

## Attribute Reference

- `id` - The login principal ID.
- `sid` - The SID (Security Identifier) of the SQL login in hexadecimal format.

## Import

Logins can be imported using the login name:

```shell
terraform import mssql_sql_login.example my_login
```
