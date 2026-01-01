# Terraform and Provider Configuration for MSSQL Resources

terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 1.0"
    }
  }
}

# MSSQL Provider - Uses Azure AD authentication
#
# Authentication Options:
# 1. Default credential chain (az login, managed identity, etc.) - use empty azure_auth {}
# 2. Service Principal - provide client_id, client_secret, tenant_id
#
# For local development, run 'az login' before applying this configuration
provider "mssql" {
  hostname = var.sql_hostname
  port     = 1433

  azure_auth {
    # Default: Uses Azure Default Credential Chain (az login, managed identity, etc.)
    # To use a Service Principal instead, uncomment and set these:
    # client_id     = var.azure_client_id
    # client_secret = var.azure_client_secret
    # tenant_id     = var.azure_tenant_id
  }
}
