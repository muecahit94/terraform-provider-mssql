---
page_title: "mssql_sql_user Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server database user mapped to a login.
---

# mssql_sql_user (Resource)

Manages a database user. It is usually mapped to a SQL Server login, which can be a SQL login, a Windows user or a
Windows group; a user can also have no login.

## Example Usage

```hcl
resource "mssql_database" "example" {
  name = "my_database"
}

resource "mssql_sql_login" "example" {
  name     = "my_login"
  password = "SecurePassword123!"
}

resource "mssql_sql_user" "example" {
  database_name  = mssql_database.example.name
  name           = "my_user"
  login_name     = mssql_sql_login.example.name
  default_schema = "dbo"
  roles          = ["db_datareader", "db_datawriter"]
}
```

### Windows Group

A login created for a Windows or Active Directory group gives a user of type `WINDOWS_GROUP`, which is read like any
other user:

```hcl
resource "mssql_sql_user" "admins" {
  database_name = mssql_database.example.name
  name          = "CORP\\db-admins"
  login_name    = "CORP\\db-admins"
  roles         = ["db_owner"]
}
```

### User Without a Login

```hcl
resource "mssql_sql_user" "no_login" {
  database_name = mssql_database.example.name
  name          = "impersonation_only"
}
```

Omitting `login_name` creates the user with `CREATE USER ... WITHOUT LOGIN`. This is also how a user whose login is gone
is represented, for example after a database was restored on another server (an orphaned user, whose SID matches no
login): it can be imported, and a configuration that omits `login_name` shows no diff.

### Map a User to Another Login, Fix an Orphaned User

Changing `login_name` runs `ALTER USER ... WITH LOGIN`: the user keeps its principal ID, permissions and role
memberships. This also fixes an **orphaned user** (for example after a database was restored on another server), whose SID
matches no login: import it without `login_name`, then set `login_name` to the login it belongs to.

```hcl
resource "mssql_sql_user" "restored" {
  database_name = "app_db"
  name          = "app_user"
  login_name    = "app_user" # was null while the user was orphaned
}
```

SQL Server cannot do this for a user that was created `WITHOUT LOGIN`, so giving such a user a login (or removing the
login of a mapped user) replaces it: the plan then shows *must be replaced*.

## Argument Reference

- `database_name` - (Required) The name of the database. Changing this forces a new resource.
- `name` - (Required) The name of the user. Changing this forces a new resource.
- `login_name` - (Optional) The name of the login to map this user to. Omit it for a user without a login (`WITHOUT LOGIN`) or an orphaned user. Changing it maps the user to the other login **in place** (`ALTER USER ... WITH LOGIN`), keeping its permissions and role memberships. Only a user created `WITHOUT LOGIN` that gets a login, or a user that loses its login, forces a new resource.
- `default_schema` - (Optional) The default schema for the user. Defaults to `dbo`.
- `roles` - (Optional) Set of database roles to assign to this user.

## Attribute Reference

- `id` - The user ID in format `database_id/principal_id`.
- `default_schema` - The default schema for the user.
- `login_name` - The login the user is mapped to, or `null` when no login matches the SID of the user.
- `roles` - The set of database roles assigned to this user.

## Import

Users can be imported using `database_name/user_name`:

```shell
terraform import mssql_sql_user.example my_database/my_user
```

