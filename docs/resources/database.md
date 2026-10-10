---
page_title: "mssql_database Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server database.
---

# mssql_database (Resource)

Manages a SQL Server database.

## Example Usage

```hcl
resource "mssql_database" "example" {
  name = "my_application_db"
}
```

### Database with a Collation, Compatibility Level and Recovery Model

```hcl
resource "mssql_database" "app" {
  name                = "app_db"
  collation           = "Ukrainian_CI_AS"
  compatibility_level = 160
  recovery_model      = "SIMPLE"
  owner_name          = "app_owner"

  # Fail instead of dropping the database when it is removed from the configuration.
  deletion_protection = true
}
```

The three settings are optional. When one is omitted the database keeps the value SQL Server assigned (taken from
the server's `model` database) and the provider reports it without planning a change, so existing configurations are
not affected.

## Argument Reference

- `name` - (Required) The name of the database. Changing this forces a new resource.
- `collation` - (Optional) The collation of the database, for example `SQL_Latin1_General_CP1_CI_AS`. Set when the database is created; defaults to the collation of the server. A collation name can only contain letters, digits and underscores. **Changing it on an existing database is rejected at plan time** instead of replacing the database: `ALTER DATABASE` does not change the collation of existing columns, and replacing the database would drop its data.
- `compatibility_level` - (Optional) The compatibility level of the database, for example `150` or `160`. Can be changed in place with `ALTER DATABASE ... SET COMPATIBILITY_LEVEL`. The levels SQL Server accepts depend on its version.
- `owner_name` - (Optional) The login that owns the database (`ALTER AUTHORIZATION ON DATABASE`). Defaults to the login that creates the database. Can be changed in place. It is empty when the owner login no longer exists.
- `deletion_protection` - (Optional) Protects the database from deletion. The provider drops a database (`SET SINGLE_USER WITH ROLLBACK IMMEDIATE`, `DROP DATABASE`) as soon as it is removed from the configuration or replaced, with all its data. While this is `true`, deleting or replacing the database fails; set it to `false` and apply before deleting. It is a setting of Terraform only and is not stored in SQL Server, so an imported database starts with `false`. Defaults to `false`.
- `recovery_model` - (Optional) The recovery model: `FULL`, `SIMPLE` or `BULK_LOGGED`. Can be changed in place with `ALTER DATABASE ... SET RECOVERY`. Switching to `FULL` or `BULK_LOGGED` does not start the log backup chain until a full backup is taken.

## Attribute Reference

- `id` - The database ID.
- `collation`, `compatibility_level`, `recovery_model`, `owner_name` - The values currently set on the database, also when they are not configured.

## Import

Databases can be imported using the database name. The collation, compatibility level and recovery model are read
from the server, so a configuration that omits them shows no diff after the import:

```shell
terraform import mssql_database.example my_application_db
```
