# =============================================================================
# Variables for MSSQL Resources
# =============================================================================

variable "sql_hostname" {
  description = "Azure SQL Server FQDN (from infrastructure output)"
  type        = string
}

variable "database_name" {
  description = "Database name (from infrastructure output)"
  type        = string
}

# =============================================================================
# Service Principal Authentication (Optional)
# =============================================================================
# If not using 'az login', provide Service Principal credentials

variable "azure_client_id" {
  description = "Azure AD Service Principal client ID (optional - uses az login if not set)"
  type        = string
  default     = ""
}

variable "azure_client_secret" {
  description = "Azure AD Service Principal client secret"
  type        = string
  sensitive   = true
  default     = ""
}

variable "azure_tenant_id" {
  description = "Azure AD tenant ID"
  type        = string
  default     = ""
}

# =============================================================================
# Azure AD User Variables
# =============================================================================

variable "developer_email" {
  description = "Email/UPN of the Azure AD user to create"
  type        = string
  default     = ""
}

variable "developer_object_id" {
  description = "Azure AD object ID of the developer"
  type        = string
  default     = ""
}

# =============================================================================
# Azure AD Service Principal Variables (for database access)
# =============================================================================

variable "app_name" {
  description = "Name of the Azure AD application to create as database user"
  type        = string
  default     = ""
}

variable "app_client_id" {
  description = "Azure AD client ID of the application"
  type        = string
  default     = ""
}

# =============================================================================
# Managed Identity Variables
# =============================================================================

variable "mi_name" {
  description = "Name of the User Assigned Managed Identity"
  type        = string
  default     = ""
}

variable "mi_object_id" {
  description = "Object ID of the User Assigned Managed Identity"
  type        = string
  default     = ""
}
