---
page_title: "mssql_windows_login Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages the login of a Windows or Active Directory user or group.
---

# mssql_windows_login (Resource)

Manages the login of a Windows or Active Directory user or group (`CREATE LOGIN ... FROM WINDOWS`). A Windows login has
no password: the account authenticates through Windows. Use [`mssql_sql_login`](sql_login.md) for SQL Server logins.

The Windows account must exist, and SQL Server must be able to resolve it (the server has to be in the domain, or the
name has to be a local one); otherwise creating the login fails.

## Example Usage

```hcl
# A user
resource "mssql_windows_login" "alice" {
  name             = "CORP\\alice"
  default_database = "master"
}

# A group: everybody in the Active Directory group can connect
resource "mssql_windows_login" "dba" {
  name = "CORP\\db-admins"
}

# A login that exists but must not be used for now
resource "mssql_windows_login" "former" {
  name        = "CORP\\bob"
  is_disabled = true
}
```

## Argument Reference

- `name` - (Required) The Windows user or group, as `DOMAIN\name` (or `name@domain`). Changing this forces a new resource.
- `default_database` - (Optional) The default database of the login. Defaults to the value SQL Server assigns (`master`). Can be changed in place.
- `default_language` - (Optional) The default language of the login. Defaults to the value SQL Server assigns. Can be changed in place.
- `is_disabled` - (Optional) Whether the login is disabled. Defaults to `false`. Can be changed in place.

## Attribute Reference

- `id` - The login principal ID.
- `type` - `WINDOWS_LOGIN` for a user, `WINDOWS_GROUP` for a group.

## Import

Windows logins can be imported using the name:

```shell
terraform import mssql_windows_login.alice 'CORP\alice'
```

The default database, language and disabled state are read from the server, so a configuration that only sets `name`
shows no diff after the import.

Deleting the resource runs `DROP LOGIN`. The database users mapped to the login stay as orphaned users.
