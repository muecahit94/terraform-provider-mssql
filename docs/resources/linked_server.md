---
page_title: "mssql_linked_server Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server linked server.
---

# mssql_linked_server (Resource)

Manages a SQL Server linked server with `sp_addlinkedserver` and `sp_serveroption`. Login mappings are managed
separately with [`mssql_linked_server_login`](linked_server_login.md).

The login that runs the provider needs the `ALTER ANY LINKED SERVER` permission (or the `setupadmin` or `sysadmin`
server role). Creating a linked server does not connect to the remote server, so the definition is created even when the
remote server is unreachable.

## Example Usage

### Linked SQL Server by Network Name

With `product = "SQL Server"` the linked server `name` is the network name of the remote instance, and no `data_source` is
needed:

```hcl
resource "mssql_linked_server" "sql_02" {
  name    = "sql-02.corp.internal"
  product = "SQL Server"
}
```

### Linked SQL Server Under a Different Name

To give the linked server a name other than the network name, use an OLE DB provider and leave `product` empty:

```hcl
resource "mssql_linked_server" "remote_sql" {
  name          = "REMOTE_SQL"
  product       = ""
  provider_name = "MSOLEDBSQL"
  data_source   = "sql-02.corp.internal"

  rpc_out = true
}
```

### Linked Server Through an OLE DB Provider

```hcl
resource "mssql_linked_server" "reporting" {
  name          = "REPORTING"
  product       = ""
  provider_name = "MSOLEDBSQL"
  data_source   = "reporting.corp.internal,1433"
  catalog       = "reports"

  data_access     = true
  rpc_out         = true
  connect_timeout = 10
  query_timeout   = 120
}
```

## Argument Reference

- `name` - (Required) The name of the linked server. Changing this forces a new resource.
- `product` - (Optional) The product name of the data source. Use `SQL Server` for a remote SQL Server and leave the provider unset. Changing this forces a new resource.
- `provider_name` - (Optional) The OLE DB provider identifier (PROGID), for example `MSOLEDBSQL` or `MSDASQL`. Named `provider_name` because `provider` is reserved by Terraform. Changing this forces a new resource.
- `data_source` - (Optional) The name of the data source as interpreted by the OLE DB provider. Changing this forces a new resource.
- `location` - (Optional) The location of the database as interpreted by the OLE DB provider. Changing this forces a new resource.
- `provider_string` - (Optional, Sensitive) The OLE DB provider-specific connection string. It commonly embeds credentials (for example `Uid`/`Pwd` of an ODBC connection string), so it is marked sensitive; it is nevertheless stored in the state, which should be encrypted. SQL Server returns it on read, so an existing linked server can be imported and the attribute left out of the configuration. Changing this forces a new resource.
- `catalog` - (Optional) The catalog or default database to use when connecting to the provider. Changing this forces a new resource.
- `rpc` - (Optional) Enables remote procedure calls from the remote server to this server. Defaults to `false`.
- `rpc_out` - (Optional) Enables remote procedure calls from this server to the remote server. Defaults to `false`.
- `data_access` - (Optional) Enables the linked server for distributed query access. Defaults to `true`.
- `collation_compatible` - (Optional) Whether the remote server has the same collation as this server. Defaults to `false`.
- `use_remote_collation` - (Optional) Whether to use the collation of the remote server instead of `collation_name`. Defaults to `true`.
- `collation_name` - (Optional) The collation to use for the remote server. Only valid when `use_remote_collation` is `false`.
- `connect_timeout` - (Optional) The connection timeout in seconds. `0` uses the server default. Defaults to `0`.
- `query_timeout` - (Optional) The query timeout in seconds. `0` uses the server default. Defaults to `0`.
- `lazy_schema_validation` - (Optional) Skips checking the schema of remote tables at the start of a query. Defaults to `false`.
- `remote_proc_transaction_promotion` - (Optional) Whether calling a remote stored procedure starts a distributed transaction. Defaults to `true`.

SQL Server cannot alter the definition of a linked server (`product`, `provider_name`, `data_source`, `location`,
`provider_string`, `catalog`), so changing one of them recreates the linked server and drops its login mappings. The
`sp_serveroption` settings are updated in place.

## Attribute Reference

- `id` - The server ID of the linked server.
- `product`, `provider_name`, `data_source`, `location`, `provider_string`, `catalog` - The values stored by SQL Server, which can differ from the configuration when SQL Server fills in defaults, for example the provider of a `SQL Server` product.

## Import

Linked servers can be imported using the linked server name:

```shell
terraform import mssql_linked_server.example REMOTE_SQL
```
