---
page_title: "mssql_sql_login Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Get information about a SQL Server login.
---

# mssql_sql_login (Data Source)

Use this data source to get information about a SQL Server login.

## Example Usage

```hcl
data "mssql_sql_login" "example" {
  name = "my_login"
}

output "is_disabled" {
  value = data.mssql_sql_login.example.is_disabled
}
```

## Argument Reference

- `name` - (Required) The name of the login.

## Attribute Reference

- `id` - The login principal ID.
- `default_database` - The default database.
- `default_language` - The default language.
- `check_expiration_enabled` - Whether password expiration is checked.
- `check_policy_enabled` - Whether password policy is enforced.
- `is_disabled` - Whether the login is disabled.
