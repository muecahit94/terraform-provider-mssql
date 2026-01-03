# =============================================================================
# MSSQL Provider Resources (Azure AD Users/Permissions)
# =============================================================================

locals {
  create_developer = var.developer_email != ""
  create_app       = var.app_name != "" && var.app_client_id != ""
}

# =============================================================================
# Azure AD User
# =============================================================================

# Create an Azure AD user
resource "mssql_azuread_user" "developer" {
  count = local.create_developer ? 1 : 0

  database_name  = var.database_name
  name           = var.developer_email
  default_schema = "dbo"
}

# Create a database role for developers
resource "mssql_database_role" "developer" {
  count = local.create_developer ? 1 : 0

  database_name = var.database_name
  name          = "developer_role"
}

# Grant permissions to the developer role (RBAC)
resource "mssql_database_permission" "developer_select" {
  count = local.create_developer ? 1 : 0

  database_name  = var.database_name
  principal_name = mssql_database_role.developer[0].name
  permission     = "SELECT"
}

resource "mssql_database_permission" "developer_execute" {
  count = local.create_developer ? 1 : 0

  database_name  = var.database_name
  principal_name = mssql_database_role.developer[0].name
  permission     = "EXECUTE"
}

# Assign the Azure AD user to the developer role
resource "mssql_database_role_member" "developer" {
  count = local.create_developer ? 1 : 0

  database_name = var.database_name
  role_name     = mssql_database_role.developer[0].name
  member_name   = mssql_azuread_user.developer[0].name
}

# =============================================================================
# Azure AD Service Principal (Application)
# =============================================================================

# Create an Azure AD service principal for the application
resource "mssql_azuread_service_principal" "app" {
  count = local.create_app ? 1 : 0

  database_name  = var.database_name
  name           = var.app_name
  client_id      = var.app_client_id
  default_schema = "dbo"
}

# Grant SELECT permission to the application
resource "mssql_database_permission" "app_select" {
  count = local.create_app ? 1 : 0

  database_name  = var.database_name
  principal_name = mssql_azuread_service_principal.app[0].name
  permission     = "SELECT"
}

# Grant EXECUTE permission to the application
resource "mssql_database_permission" "app_execute" {
  count = local.create_app ? 1 : 0

  database_name  = var.database_name
  principal_name = mssql_azuread_service_principal.app[0].name
  permission     = "EXECUTE"
}

# =============================================================================
# User Assigned Managed Identity
# =============================================================================

locals {
  create_uami = var.mi_name != "" && var.mi_object_id != ""
}

# Create a database user for the Managed Identity
resource "mssql_azuread_user" "uami" {
  count = local.create_uami ? 1 : 0

  database_name  = var.database_name
  name           = var.mi_name
  object_id      = var.mi_object_id
  default_schema = "dbo"
}

# Create a database role for the Managed Identity
resource "mssql_database_role" "mi_role" {
  count = local.create_uami ? 1 : 0

  database_name = var.database_name
  name          = "managed_identity_role"
}

# Grant SELECT permission to the MI role
resource "mssql_database_permission" "mi_role_select" {
  count = local.create_uami ? 1 : 0

  database_name  = var.database_name
  principal_name = mssql_database_role.mi_role[0].name
  permission     = "SELECT"
}

# Assign the Managed Identity user to the MI role
resource "mssql_database_role_member" "mi_role_member" {
  count = local.create_uami ? 1 : 0

  database_name = var.database_name
  role_name     = mssql_database_role.mi_role[0].name
  member_name   = mssql_azuread_user.uami[0].name
}
