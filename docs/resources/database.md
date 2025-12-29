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

## Argument Reference

- `name` - (Required) The name of the database. Changing this forces a new resource.

## Attribute Reference

- `id` - The database ID.

## Import

Databases can be imported using the database name:

```shell
terraform import mssql_database.example my_application_db
```
