---
page_title: "mssql_script Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Executes custom SQL scripts for create, read, update, and delete operations.
---

# mssql_script (Resource)

Executes custom SQL scripts. Useful for managing resources not covered by other resources.

## Example Usage

```hcl
resource "mssql_script" "example" {
  database_name = mssql_database.example.name
  
  create_script = <<-SQL
    CREATE TABLE dbo.example (
      id INT PRIMARY KEY IDENTITY(1,1),
      name NVARCHAR(100)
    )
  SQL
  
  read_script = <<-SQL
    SELECT 
      OBJECT_ID('dbo.example') as table_id,
      'example' as table_name
  SQL
  
  delete_script = <<-SQL
    DROP TABLE IF EXISTS dbo.example
  SQL
}
```

## Argument Reference

- `database_name` - (Optional) The database to execute scripts in.
- `create_script` - (Required) SQL script to execute on resource creation.
- `read_script` - (Optional) SQL script to execute on resource read. Should return a single row.
- `update_script` - (Optional) SQL script to execute on resource update.
- `delete_script` - (Required) SQL script to execute on resource deletion.

## Attribute Reference

- `state` - A map of values returned from the read script.
