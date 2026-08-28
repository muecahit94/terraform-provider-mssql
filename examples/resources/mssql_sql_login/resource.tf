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

# Login with a write-only password, so it is stored in neither the plan nor the
# state file (requires Terraform >= 1.11).
ephemeral "random_password" "generated" {
  length           = 32
  override_special = "!#$*()-_+[]{}<>?"
}

resource "mssql_sql_login" "write_only_password" {
  name        = "ephemeral_login"
  password_wo = ephemeral.random_password.generated.result

  # Terraform cannot compare a write-only value against state; bump this token
  # to apply a rotated password.
  password_wo_version = "1"
}
