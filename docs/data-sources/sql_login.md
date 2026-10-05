---
page_title: "mssql_sql_login Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Get information about a SQL Server login.
---

# mssql_sql_login (Data Source)

Use this data source to get information about a SQL Server login.

## Example Usage

### Default Provider Connection

```hcl
data "mssql_sql_login" "example" {
  name = "my_login"
}

output "is_disabled" {
  value = data.mssql_sql_login.example.is_disabled
}
```

### With Server Override and login_name Alias

```hcl
data "mssql_sql_login" "cluster_user" {
  server {
    hostname = "sql-cluster.internal"
    port     = 1433
    sql_auth {
      username = "sa"
      password = "SecretPassword123!"
    }
  }

  login_name = "app_user"
}

output "login_sid" {
  value = data.mssql_sql_login.cluster_user.sid
}
```

### Dynamic Multi-Server Lookup (for_each)

```hcl
locals {
  servers = {
    "node1" = { host = "sql-node1.internal", sa_user = "sa", sa_pass = "SecretPassword123!" }
    "node2" = { host = "sql-node2.internal", sa_user = "sa", sa_pass = "SecretPassword123!" }
  }
}

data "mssql_sql_login" "cluster_users" {
  for_each = local.servers

  server {
    host = each.value.host
    port = 1433
    login {
      username = each.value.sa_user
      password = each.value.sa_pass
    }
  }

  login_name = "app_user"
}
```

## Argument Reference

- `name` - (Optional) The name of the login. Exactly one of `name` or `login_name` must be set.
- `login_name` - (Optional) Alias for `name`. The name of the login. Exactly one of `name` or `login_name` must be set.
- `server` - (Optional) SQL Server instance configuration block. When omitted, the data source uses the default provider-level connection.
  - `hostname` - (Optional) FQDN or IP address of the target SQL endpoint.
  - `host` - (Optional) Alias for `hostname`.
  - `port` - (Optional) TCP port of SQL endpoint. Defaults to `1433`.
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
- `default_database` - The default database.
- `default_language` - The default language.
- `check_expiration_enabled` - Whether password expiration is checked.
- `check_policy_enabled` - Whether password policy is enforced.
- `is_disabled` - Whether the login is disabled.
