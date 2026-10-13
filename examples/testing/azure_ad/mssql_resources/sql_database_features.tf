# ============================================================================
# Features that must work on Azure SQL Database
# ============================================================================

# Reading a database: Azure SQL Database rejects SUSER_SNAME(sid) (issue #49)
data "mssql_database" "this" {
  name = var.database_name
}

data "mssql_databases" "all" {}

# SQL logins live in master; the user is created in the database.
resource "random_password" "sql_login" {
  length           = 24
  min_upper        = 2
  min_lower        = 2
  min_numeric      = 2
  min_special      = 2
  override_special = "!#%*-_"
}

resource "mssql_sql_login" "a" {
  name     = "e2e_login_a"
  password = random_password.sql_login.result
}

resource "mssql_sql_login" "b" {
  name     = "e2e_login_b"
  password = random_password.sql_login.result
}

# Switching sql_user_login to e2e_login_b maps the user to the other login in place (ALTER USER WITH LOGIN).
resource "mssql_sql_user" "app" {
  database_name = var.database_name
  name          = "e2e_sql_user"
  login_name    = var.sql_user_login == mssql_sql_login.b.name ? mssql_sql_login.b.name : mssql_sql_login.a.name
}

resource "mssql_sql_user" "no_login" {
  database_name = var.database_name
  name          = "e2e_no_login"
}

resource "mssql_script" "orders" {
  database_name = var.database_name
  create_script = "CREATE TABLE dbo.e2e_orders (id int PRIMARY KEY, total money)"
  delete_script = "DROP TABLE IF EXISTS dbo.e2e_orders"
}

resource "mssql_database_object_permission" "orders_select" {
  database_name  = var.database_name
  schema_name    = "dbo"
  object_name    = "e2e_orders"
  principal_name = mssql_sql_user.app.name
  permission     = "SELECT"
  depends_on     = [mssql_script.orders]
}

resource "mssql_database_object_permission" "orders_total_update" {
  database_name  = var.database_name
  schema_name    = "dbo"
  object_name    = "e2e_orders"
  column_name    = "total"
  principal_name = mssql_sql_user.app.name
  permission     = "UPDATE"
  depends_on     = [mssql_script.orders]
}

output "database_owner" {
  value = data.mssql_database.this.owner_name
}

output "database_count" {
  value = length(data.mssql_databases.all.databases)
}
