resource "mssql_linked_server" "remote_sql" {
  name          = "REMOTE_SQL"
  product       = ""
  provider_name = "MSOLEDBSQL"
  data_source   = "sql-02.corp.internal"
}

# Map all local logins to one remote login
resource "mssql_linked_server_login" "all" {
  server_name = mssql_linked_server.remote_sql.name
  remote_user = "linked_reader"
  password    = "SecretPassword123!"
}

# Map one local login to a remote login, with a write-only password so it is
# stored in neither the plan nor the state file (requires Terraform >= 1.11).
ephemeral "random_password" "remote" {
  length           = 32
  override_special = "!#$*()-_+[]{}<>?"
}

resource "mssql_linked_server_login" "app" {
  server_name = mssql_linked_server.remote_sql.name
  local_login = "app_login"
  remote_user = "linked_app"
  password_wo = ephemeral.random_password.remote.result

  # Terraform cannot compare a write-only value against state; bump this token
  # to apply a rotated password.
  password_wo_version = "1"
}

# Let a local login connect with its own credentials
resource "mssql_linked_server_login" "pass_through" {
  server_name = mssql_linked_server.remote_sql.name
  local_login = "admin_login"
  use_self    = true
}
