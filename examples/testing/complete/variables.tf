variable "sql_hostname" {
  description = "SQL Server hostname"
  type        = string
  default     = "localhost"
}

variable "sql_port" {
  description = "SQL Server port"
  type        = number
  default     = 1433
}

variable "sql_username" {
  description = "SQL Server admin username"
  type        = string
  default     = "sa"
}

variable "sql_password" {
  description = "SQL Server admin password"
  type        = string
  sensitive   = true
}

variable "app_password" {
  description = "Application login password"
  type        = string
  sensitive   = true
}
