---
page_title: "mssql_linked_server_login Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages the mapping of a local login to a remote login on a SQL Server linked server.
---

# mssql_linked_server_login (Resource)

Manages a login mapping of a linked server with `sp_addlinkedsrvlogin`. A mapping tells SQL Server which remote login a
local login uses when it queries the linked server.

## Example Usage

### Map All Local Logins to One Remote Login

```hcl
resource "mssql_linked_server_login" "all" {
  server_name = mssql_linked_server.remote_sql.name
  remote_user = "linked_reader"
  password    = "SecretPassword123!"
}
```

### Map One Local Login with a Write-Only Password

`password_wo` is a [write-only attribute](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only): it accepts
[ephemeral](https://developer.hashicorp.com/terraform/language/resources/ephemeral) values and Terraform writes it to neither the plan nor
the state file. It requires Terraform 1.11 or later.

```hcl
ephemeral "random_password" "remote" {
  length           = 32
  override_special = "!#$*()-_+[]{}<>?"
}

resource "mssql_linked_server_login" "app" {
  server_name         = mssql_linked_server.remote_sql.name
  local_login         = "app_login"
  remote_user         = "linked_app"
  password_wo         = ephemeral.random_password.remote.result
  password_wo_version = "1"
}
```

Because the value is not stored, Terraform cannot detect that it changed. The provider re-applies the mapping only when
`password_wo_version` changes, so bump it in the same apply that rotates the password. A write-only password never reaches
the state, so nothing can read it back afterwards: write it to a secret store in the same apply, or consumers will have no
way to obtain it.

### Connect With the Caller's Own Credentials

```hcl
resource "mssql_linked_server_login" "pass_through" {
  server_name = mssql_linked_server.remote_sql.name
  local_login = "admin_login"
  use_self    = true
}
```

## Argument Reference

- `server_name` - (Required) The name of the linked server. Changing this forces a new resource.
- `local_login` - (Optional) The local login the mapping applies to. If omitted, the mapping applies to all local logins. Changing this forces a new resource.
- `use_self` - (Optional) Whether the local login connects with its own credentials instead of a remote login. Defaults to `false`. When `true`, `remote_user` and the password attributes cannot be set.
- `remote_user` - (Optional) The remote login to connect as. Required unless `use_self` is `true`.
- `password` - (Optional) The password of the remote login. Persisted in the plan and state files. Exactly one of `password` and `password_wo` must be set unless `use_self` is `true`.
- `password_wo` - (Optional, [write-only](https://developer.hashicorp.com/terraform/language/resources/ephemeral/write-only)) The password of the remote login. Accepts ephemeral values and is written to neither the plan nor the state file. Requires Terraform 1.11 or later.
- `password_wo_version` - (Optional) An arbitrary token whose change re-applies the mapping with the current `password_wo` value. Only valid together with `password_wo`. Without it, a rotated `password_wo` is never applied.

SQL Server never returns the remote password, so a password changed outside Terraform is not detected. A changed
`remote_user` or `use_self` is detected and corrected on the next apply.

Removing the mapping for all local logins (no `local_login`) also removes the default mapping that SQL Server creates for a
new linked server, so local logins that have no mapping of their own cannot use the linked server until another mapping
exists.

## Attribute Reference

- `id` - The resource ID, in the form `server_name/local_login`. The local login is empty for a mapping that applies to all local logins.

## Import

Login mappings can be imported using `server_name/local_login`. Leave `local_login` empty for the mapping that applies to
all local logins:

```shell
terraform import mssql_linked_server_login.app REMOTE_SQL/app_login
terraform import mssql_linked_server_login.all REMOTE_SQL/
```

The remote password is not imported. Set `password` or `password_wo` in the configuration; the next apply writes it to
the server.
