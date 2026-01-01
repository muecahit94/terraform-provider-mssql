# =============================================================================
# Azure SQL Server Infrastructure
# =============================================================================

# Data sources
data "azurerm_client_config" "current" {}

data "azuread_user" "current" {
  count     = var.enable_azure_ad_admin ? 1 : 0
  object_id = data.azurerm_client_config.current.object_id
}

data "http" "my_ip" {
  url = "https://ifconfig.me/ip"
}

# Random Password for SQL Admin (if not provided)
resource "random_password" "admin_password" {
  count       = var.sql_admin_password == "" ? 1 : 0
  length      = 16
  special     = true
  min_upper   = 1
  min_lower   = 1
  min_numeric = 1
  min_special = 1
}

locals {
  admin_password = var.sql_admin_password != "" ? var.sql_admin_password : random_password.admin_password[0].result
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = var.resource_group_name
  location = var.location
}

# User Assigned Managed Identity
resource "azurerm_user_assigned_identity" "example" {
  location            = azurerm_resource_group.main.location
  name                = "uami-mssql-test"
  resource_group_name = azurerm_resource_group.main.name
}

# Azure SQL Server
resource "azurerm_mssql_server" "main" {
  name                         = var.sql_server_name
  resource_group_name          = azurerm_resource_group.main.name
  location                     = azurerm_resource_group.main.location
  version                      = "12.0"
  administrator_login          = var.sql_admin_username
  administrator_login_password = local.admin_password
  minimum_tls_version          = "1.2"

  # Enable Azure AD authentication
  dynamic "azuread_administrator" {
    for_each = var.enable_azure_ad_admin ? [1] : []
    content {
      login_username              = data.azuread_user.current[0].user_principal_name
      object_id                   = data.azurerm_client_config.current.object_id
      tenant_id                   = data.azurerm_client_config.current.tenant_id
      azuread_authentication_only = false
    }
  }

  tags = {
    environment = "test"
    purpose     = "mssql-provider-testing"
  }
}

# Firewall Rules
resource "azurerm_mssql_firewall_rule" "allow_azure_services" {
  name             = "AllowAzureServices"
  server_id        = azurerm_mssql_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

resource "azurerm_mssql_firewall_rule" "allow_my_ip" {
  name             = "AllowMyIP"
  server_id        = azurerm_mssql_server.main.id
  start_ip_address = data.http.my_ip.response_body
  end_ip_address   = data.http.my_ip.response_body
}

# Azure SQL Database
resource "azurerm_mssql_database" "main" {
  name           = var.database_name
  server_id      = azurerm_mssql_server.main.id
  collation      = "SQL_Latin1_General_CP1_CI_AS"
  license_type   = "LicenseIncluded"
  max_size_gb    = 2
  sku_name       = "Basic"
  zone_redundant = false

  tags = {
    environment = "test"
    purpose     = "mssql-provider-testing"
  }
}
