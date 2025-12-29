output "database_id" {
  description = "The database ID"
  value       = mssql_database.app.id
}

output "login_id" {
  description = "The login principal ID"
  value       = mssql_sql_login.app.id
}

output "user_id" {
  description = "The user ID"
  value       = mssql_sql_user.app.id
}
