output "database_name" {
  description = "The database name"
  value       = mssql_database.app.name
}

output "login_name" {
  description = "The login principal name"
  value       = mssql_sql_login.app.name
}

output "user_name" {
  description = "The user name"
  value       = mssql_sql_user.app.name
}

output "sid_login_sid" {
  description = "The SID of the custom SID login"
  value       = data.mssql_sql_login.sid_login.sid
}
