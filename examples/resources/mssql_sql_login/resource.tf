# Basic login (automatic SID)
resource "mssql_sql_login" "example" {
  name     = "example_login"
  password = "SecretPassword123!"
}

# Login with custom SID (e.g. for Availability Groups / server mirroring)
resource "mssql_sql_login" "with_sid" {
  name     = "mirrored_login"
  password = "SecretPassword123!"
  sid      = "0x0123456789ABCDEF0123456789ABCDEF"
}
