# Limit the memory of SQL Server (an advanced option: "show advanced options" is switched on for the
# change and off again by the provider)
resource "mssql_server_configuration" "max_memory" {
  name  = "max server memory (MB)"
  value = 8192
}

# Keep the operating system shell closed to T-SQL
resource "mssql_server_configuration" "xp_cmdshell" {
  name  = "xp_cmdshell"
  value = 0
}

# An option that takes effect only after a restart reports it
resource "mssql_server_configuration" "worker_threads" {
  name  = "max worker threads"
  value = 640
}

output "worker_threads_need_restart" {
  value = mssql_server_configuration.worker_threads.restart_required
}
