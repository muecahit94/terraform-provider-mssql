---
page_title: "mssql_database_role Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server database role.
---

# mssql_database_role (Resource)

Manages a custom database role.

## Example Usage

```hcl
resource "mssql_database" "example" {
  name = "my_database"
}

resource "mssql_database_role" "example" {
  database_name = mssql_database.example.name
  name          = "app_readers"
}
```

## Argument Reference

- `database_name` - (Required) The name of the database. Changing this forces a new resource.
- `name` - (Required) The name of the role. Changing this forces a new resource.
- `owner_name` - (Optional) The owner of the role.

## Attribute Reference

- `id` - The role ID in format `database_id/principal_id`.

## Import

```shell
terraform import mssql_database_role.example my_database/app_readers
```
