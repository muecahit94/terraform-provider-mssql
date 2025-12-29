---
page_title: "mssql_query Data Source - terraform-provider-mssql"
subcategory: ""
description: |-
  Execute a custom SQL query and return the results.
---

# mssql_query (Data Source)

Use this data source to execute a custom SQL query and retrieve results.

## Example Usage

```hcl
data "mssql_query" "version" {
  query = "SELECT @@VERSION as version"
}

output "sql_version" {
  value = data.mssql_query.version.result[0].values["version"]
}

data "mssql_query" "tables" {
  database_name = "my_database"
  query         = "SELECT TABLE_SCHEMA, TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"
}
```

## Argument Reference

- `database_name` - (Optional) The database to execute the query in.
- `query` - (Required) The SQL query to execute.

## Attribute Reference

- `result` - A list of rows, each with:
  - `values` - A map of column names to values.
