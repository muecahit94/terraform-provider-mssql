# =============================================================================
# Outputs
# =============================================================================

output "sql_server_fqdn" {
  description = "FQDN of the Azure SQL Server - use this as sql_hostname in mssql_resources"
  value       = azurerm_mssql_server.main.fully_qualified_domain_name
}

output "sql_server_name" {
  description = "Name of the Azure SQL Server"
  value       = azurerm_mssql_server.main.name
}

output "database_name" {
  description = "Name of the database"
  value       = azurerm_mssql_database.main.name
}

output "sql_admin_username" {
  description = "SQL admin username"
  value       = var.sql_admin_username
}

output "sql_admin_password" {
  description = "SQL admin password"
  value       = local.admin_password
  sensitive   = true
}

output "resource_group_name" {
  description = "Name of the resource group"
  value       = azurerm_resource_group.main.name
}

output "connection_string" {
  description = "Connection string for the database (without password)"
  value       = "Server=${azurerm_mssql_server.main.fully_qualified_domain_name};Database=${azurerm_mssql_database.main.name};User ID=${var.sql_admin_username};Password=${local.admin_password};Encrypt=True;TrustServerCertificate=False;"
  sensitive   = true
}

output "mi_name" {
  description = "Name of the User Assigned Managed Identity"
  value       = azurerm_user_assigned_identity.example.name
}

output "mi_client_id" {
  description = "Client ID of the User Assigned Managed Identity"
  value       = azurerm_user_assigned_identity.example.client_id
}

output "mi_principal_id" {
  description = "Principal ID (Object ID) of the User Assigned Managed Identity"
  value       = azurerm_user_assigned_identity.example.principal_id
}

output "next_step" {
  description = "Instructions for the next step"
  value       = <<-EOT

    Infrastructure created successfully!

    Next step: Deploy MSSQL resources

    cd ../mssql_resources

    # Create terraform.tfvars with:
    sql_hostname = "${azurerm_mssql_server.main.fully_qualified_domain_name}"
    sql_hostname = "${azurerm_mssql_server.main.fully_qualified_domain_name}"
    database_name = "${azurerm_mssql_database.main.name}"
    mi_name = "${azurerm_user_assigned_identity.example.name}"
    mi_client_id = "${azurerm_user_assigned_identity.example.client_id}"
    mi_object_id = "${azurerm_user_assigned_identity.example.principal_id}"

    terraform init
    terraform apply
  EOT
}
