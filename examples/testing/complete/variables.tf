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

variable "wo_password" {
  description = "Password for the login that uses the write-only password_wo attribute"
  type        = string
  sensitive   = true
  default     = "WriteOnlyP@ssw0rd123!"
}

variable "wo_password_version" {
  description = "Rotation token for wo_password; changing it applies the current value"
  type        = string
  default     = "1"
}
