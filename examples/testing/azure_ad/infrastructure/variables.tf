# =============================================================================
# Azure Infrastructure Variables
# =============================================================================

variable "subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "resource_group_name" {
  description = "Name of the resource group"
  type        = string
  default     = "rg-mssql-provider-test"
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "westeurope"
}

variable "sql_server_name" {
  description = "Name of the Azure SQL Server (must be globally unique)"
  type        = string
  default     = "sql-mssql-provider-test-123"
}

variable "sql_admin_username" {
  description = "SQL Server administrator username"
  type        = string
  default     = "sqladmin"
}

variable "sql_admin_password" {
  description = "SQL Server administrator password"
  type        = string
  default     = ""
  sensitive   = true
}

variable "database_name" {
  description = "Name of the database to create"
  type        = string
  default     = "testdb"
}

variable "enable_azure_ad_admin" {
  description = "Enable Azure AD admin for the SQL Server"
  type        = bool
  default     = true
}
