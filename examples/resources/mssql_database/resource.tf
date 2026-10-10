# Database with the collation, compatibility level and recovery model of the server's model database
resource "mssql_database" "default" {
  name = "default_db"
}

# Database with explicit settings. The collation is set at creation; the compatibility
# level and the recovery model can later be changed in place.
resource "mssql_database" "example" {
  name                = "example_db"
  collation           = "SQL_Latin1_General_CP1_CI_AS"
  compatibility_level = 160
  recovery_model      = "SIMPLE"
  owner_name          = "app_owner"

  # Fail instead of dropping the database when it is removed from the configuration.
  deletion_protection = true

  page_verify             = "CHECKSUM"
  auto_shrink             = false
  read_committed_snapshot = true
}
