# Linked server named after the network name of a remote SQL Server instance
resource "mssql_linked_server" "sql_02" {
  name    = "sql-02.corp.internal"
  product = "SQL Server"
}

# Linked server to another SQL Server instance under a different name
resource "mssql_linked_server" "remote_sql" {
  name          = "REMOTE_SQL"
  product       = ""
  provider_name = "MSOLEDBSQL"
  data_source   = "sql-02.corp.internal"

  rpc_out = true
}

# Linked server through an OLE DB provider with a catalog and timeouts
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

# Linked server with its own collation instead of the remote one
resource "mssql_linked_server" "legacy" {
  name          = "LEGACY"
  provider_name = "MSOLEDBSQL"
  data_source   = "legacy.corp.internal"

  use_remote_collation = false
  collation_name       = "Latin1_General_CI_AS"
}
