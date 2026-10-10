# A user
resource "mssql_windows_login" "alice" {
  name             = "CORP\\alice"
  default_database = "master"
}

# A group: everybody in the Active Directory group can connect
resource "mssql_windows_login" "dba" {
  name = "CORP\\db-admins"
}

# A login that exists but must not be used for now
resource "mssql_windows_login" "former" {
  name        = "CORP\\bob"
  is_disabled = true
}
