# =============================================================================
# Outputs
# =============================================================================

output "azuread_user_name" {
  description = "Name of the created Azure AD user"
  value       = try(mssql_azuread_user.developer[0].name, null)
}

output "azuread_service_principal_name" {
  description = "Name of the created Azure AD service principal"
  value       = try(mssql_azuread_service_principal.app[0].name, null)
}

output "group_user_type" {
  description = "Type of the Azure AD group user"
  value       = try(mssql_azuread_user.group[0].type, null)
}
