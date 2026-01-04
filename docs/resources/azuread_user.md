---
page_title: "mssql_azuread_user Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages an Azure AD user in a SQL Server database.
---

# mssql_azuread_user (Resource)

Manages an Azure AD user in a database. Requires Azure AD authentication.

## Example Usage

### Email-based user with roles

```hcl
resource "mssql_azuread_user" "email_user" {
  database_name  = mssql_database.example.name
  name           = "john.doe@contoso.com"
  default_schema = "dbo"
  roles          = ["db_datareader", "db_datawriter"]
}
```

### Managed Identity (object_id required)

```hcl
resource "mssql_azuread_user" "managed_identity" {
  database_name  = mssql_database.example.name
  name           = "my-managed-identity"
  object_id      = "00000000-0000-0000-0000-000000000000"
  default_schema = "dbo"
  roles          = ["db_datareader"]
}
```

## Argument Reference

- `database_name` - (Required) The name of the database.
- `name` - (Required) The display name of the Azure AD user.
- `object_id` - (Optional) The Azure AD object ID of the user. Required for managed identities, optional for email-based users. When not provided, the user is created using `FROM EXTERNAL PROVIDER`.
- `default_schema` - (Optional) The default schema for the user. Defaults to `dbo`.
- `roles` - (Optional) Set of database roles to assign to this user.

## Attribute Reference

- `id` - The user ID in format `database_id/principal_id`.
- `object_id` - The Azure AD object ID (if provided).
- `default_schema` - The default schema for the user.
- `roles` - The set of database roles assigned to this user.

## Import

```shell
terraform import mssql_azuread_user.example my_database/john.doe@contoso.com
```
