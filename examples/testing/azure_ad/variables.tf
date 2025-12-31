variable "sql_hostname" {
  description = "Azure SQL server hostname"
  type        = string
}

variable "database_name" {
  description = "Database name"
  type        = string
}

variable "azure_client_id" {
  description = "Azure AD Service Principal client ID"
  type        = string
}

variable "azure_client_secret" {
  description = "Azure AD Service Principal client secret"
  type        = string
  sensitive   = true
}

variable "azure_tenant_id" {
  description = "Azure AD tenant ID"
  type        = string
}

variable "developer_email" {
  description = "Developer email (Azure AD user principal name)"
  type        = string
}

variable "developer_object_id" {
  description = "Developer Azure AD object ID"
  type        = string
}

variable "app_name" {
  description = "Application service principal display name"
  type        = string
}

variable "app_client_id" {
  description = "Application client ID"
  type        = string
}
