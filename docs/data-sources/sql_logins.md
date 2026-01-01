---
page_title: "mssql_sql_logins Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Get information about all SQL Server logins.
---

# mssql_sql_logins (Data Source)

Use this data source to list all SQL logins on the server.

## Example Usage

```hcl
data "mssql_sql_logins" "all" {}

output "login_names" {
  value = [for login in data.mssql_sql_logins.all.logins : login.name]
}
```

## Argument Reference

This data source has no required arguments.

## Attribute Reference

- `logins` - A list of logins with all login attributes.
