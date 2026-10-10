# Read a table
resource "mssql_database_object_permission" "orders_select" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "Orders"
  principal_name = mssql_database_role.readers.name
  permission     = "SELECT"
}

# Update one column only
resource "mssql_database_object_permission" "orders_total_update" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "Orders"
  column_name    = "Total"
  principal_name = mssql_sql_user.app.name
  permission     = "UPDATE"
}

# Run a stored procedure
resource "mssql_database_object_permission" "close_execute" {
  database_name  = mssql_database.app.name
  schema_name    = "sales"
  object_name    = "usp_CloseOrders"
  principal_name = mssql_sql_user.app.name
  permission     = "EXECUTE"
}

# See the definition, and let the principal pass the permission on
resource "mssql_database_object_permission" "orders_definition" {
  database_name     = mssql_database.app.name
  schema_name       = "sales"
  object_name       = "Orders"
  principal_name    = mssql_sql_user.app.name
  permission        = "VIEW DEFINITION"
  with_grant_option = true
}
