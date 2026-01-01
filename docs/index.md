---
page_title: "Provider: MSSQL"
description: |-
  The MSSQL provider allows you to manage Microsoft SQL Server and Azure SQL resources.
---

# MSSQL Provider

A Terraform provider for managing Microsoft SQL Server and Azure SQL resources.

## Features

- Manage databases, logins, users, roles, schemas, and permissions
- Support for Azure SQL Database and Managed Instance
- Azure AD authentication (service principals and managed identities)
- Resilient design that handles ID changes gracefully

## Example Usage

```hcl
terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 1.0"
    }
  }
}

# SQL Authentication
provider "mssql" {
  hostname = "localhost"
  port     = 1433

  sql_auth {
    username = "sa"
    password = var.sa_password
  }
}

# Create resources
resource "mssql_database" "example" {
  name = "my_database"
}
```

## Authentication

### SQL Authentication

```hcl
provider "mssql" {
  hostname = "myserver.example.com"
  port     = 1433

  sql_auth {
    username = "sa"
    password = "YourPassword123!"
  }
}
```

### Azure AD - Service Principal

```hcl
provider "mssql" {
  hostname = "myserver.database.windows.net"
  port     = 1433

  azure_auth {
    client_id     = "00000000-0000-0000-0000-000000000000"
    client_secret = "your-secret"
    tenant_id     = "00000000-0000-0000-0000-000000000000"
  }
}
```

### Azure AD - Default Credentials

Uses the Azure credential chain (environment variables, managed identity, Azure CLI, etc.):

```hcl
provider "mssql" {
  hostname   = "myserver.database.windows.net"
  port       = 1433
  azure_auth {}
}
```

## Schema

### Optional

- `hostname` (String) SQL Server hostname. Can be set via `MSSQL_HOSTNAME` environment variable.
- `port` (Number) SQL Server port. Defaults to `1433`. Can be set via `MSSQL_PORT` environment variable.

### Blocks

#### sql_auth

SQL Server authentication credentials.

- `username` (String, Required) Username.
- `password` (String, Required, Sensitive) Password.

#### azure_auth

Azure AD authentication. When set to empty block `{}`, uses default credential chain.

- `client_id` (String, Optional) Service principal client ID.
- `client_secret` (String, Optional, Sensitive) Service principal secret.
- `tenant_id` (String, Optional) Azure AD tenant ID.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `MSSQL_HOSTNAME` | SQL Server hostname |
| `MSSQL_PORT` | SQL Server port |
| `ARM_CLIENT_ID` | Azure AD client ID |
| `ARM_CLIENT_SECRET` | Azure AD client secret |
| `ARM_TENANT_ID` | Azure AD tenant ID |
