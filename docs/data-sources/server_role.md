---
page_title: "mssql_server_role Data Source - terraform-provider-mssql"
description: |-
  Use this data source to get information about a server role.
---

# mssql_server_role (Data Source)

Use this data source to get information about a server role.

## Example Usage

```hcl
data "mssql_server_role" "example" {
  name = "sysadmin"
}

output "owner" {
  value = data.mssql_server_role.example.owner_name
}
```

## Argument Reference

- `name` - (Required) The name of the server role.

## Attribute Reference

- `id` - The principal ID of the server role.
- `owner_name` - The name of the role owner.
