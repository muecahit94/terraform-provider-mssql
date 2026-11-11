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

### Named SQL Server Connections

Use the optional `servers` map to manage SQL logins on multiple nodes with one
provider configuration. Each login selects an entry using `server_key`.

```hcl
variable "nodes" {
  type = map(object({
    hostname = string
    port     = optional(number, 1433)
  }))
}

variable "node_credentials" {
  type = map(object({
    username = string
    password = string
  }))
  sensitive = true
}

provider "mssql" {
  servers = {
    for key, node in var.nodes : key => {
      hostname = node.hostname
      port     = node.port
      username = var.node_credentials[key].username
      password = var.node_credentials[key].password
    }
  }
}

resource "mssql_sql_login" "app" {
  for_each = var.nodes

  server_key = each.key
  name       = "app_login"
  password   = var.app_password
}
```

Server keys, hostnames and credentials must be non-empty and known when the
provider is configured. Port defaults to `1433` and must be between `1` and
`65535`. Endpoints must be unique; matching ignores hostname case and a trailing
dot, but does not resolve DNS aliases.

The registry supports SQL authentication for `mssql_sql_login` resources and data
sources only. Other resources and data sources still need a default provider
connection or a separately configured provider alias. A registry can coexist with
a default connection; it does not select a default server automatically.

Credentials come from the current provider configuration for each connection,
not the login's resource state. The map is not a background secret refresh:
Terraform must evaluate updated credentials when configuring the provider.
Passwords are sensitive, not write-only; protect saved plan files and any
credential data sources that persist secrets in state.

Existing default-provider and resource-level authentication remain unchanged
when `servers` is omitted. When it is configured, legacy resource `server`
blocks must match a registered endpoint and use registry credentials instead.
See the [login migration instructions](resources/sql_login.md#multi-host-logins-with-provider-credentials)
before switching existing resources.

## Schema

### Optional

- `hostname` (String) SQL Server hostname. Can be set via `MSSQL_HOSTNAME` environment variable.
- `port` (Number) SQL Server port. Defaults to `1433`. Can be set via `MSSQL_PORT` environment variable.
- `servers` (Map of Object) Named SQL Server connections for `mssql_sql_login` resources and data sources. Each object requires `hostname`, `username`, and sensitive `password`; `port` is optional and defaults to `1433`.

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
