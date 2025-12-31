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
