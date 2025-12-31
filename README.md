# Terraform Provider for Microsoft SQL Server

A Terraform provider to manage Microsoft SQL Server and Azure SQL resources.

## Features

- **Full SQL Server Support**: Manage databases, logins, users, roles, schemas, and permissions
- **Azure SQL Compatible**: Works with Azure SQL Database and Managed Instance
- **Azure AD Authentication**: Support for service principals and managed identities
- **Resilient Design**: Gracefully handles ID changes and manual modifications

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- Go >= 1.21 (for building from source)
- SQL Server 2016+ or Azure SQL

## Installation

```hcl
terraform {
  required_providers {
    mssql = {
      source  = "muecahit94/mssql"
      version = "~> 1.0"
    }
  }
}
```

## Configuration

### SQL Authentication

```hcl
provider "mssql" {
  hostname = "localhost"
  port     = 1433

  sql_auth {
    username = "sa"
    password = "YourPassword123!"
  }
}
```

### Azure AD Authentication

```hcl
# Using Service Principal
provider "mssql" {
  hostname = "myserver.database.windows.net"
  port     = 1433

  azure_auth {
    client_id     = "00000000-0000-0000-0000-000000000000"
    client_secret = "your-secret"
    tenant_id     = "00000000-0000-0000-0000-000000000000"
  }
}

# Using Default Azure Credentials (Managed Identity, Azure CLI, etc.)
provider "mssql" {
  hostname   = "myserver.database.windows.net"
  port       = 1433
  azure_auth {}
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `MSSQL_HOSTNAME` | SQL Server hostname |
| `MSSQL_PORT` | SQL Server port |
| `ARM_CLIENT_ID` | Azure AD client ID |
| `ARM_CLIENT_SECRET` | Azure AD client secret |
| `ARM_TENANT_ID` | Azure AD tenant ID |

## Resources

| Resource | Description |
|----------|-------------|
| `mssql_database` | SQL Server database |
| `mssql_sql_login` | SQL Server login |
| `mssql_sql_user` | Database user mapped to login |
| `mssql_database_role` | Database role |
| `mssql_database_role_member` | Database role membership |
| `mssql_database_permission` | Database-level permission |
| `mssql_schema` | Database schema |
| `mssql_schema_permission` | Schema-level permission |
| `mssql_server_role` | Server role |
| `mssql_server_role_member` | Server role membership |
| `mssql_server_permission` | Server-level permission |
| `mssql_script` | Custom SQL script execution |
| `mssql_azuread_user` | Azure AD user |
| `mssql_azuread_service_principal` | Azure AD service principal |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `mssql_database` | Get database info |
| `mssql_databases` | List all databases |
| `mssql_sql_login` | Get login info |
| `mssql_sql_logins` | List all logins |
| `mssql_sql_user` | Get user info |
| `mssql_sql_users` | List database users |
| `mssql_database_role` | Get role info |
| `mssql_database_roles` | List database roles |
| `mssql_database_permissions` | Get database permissions |
| `mssql_schema` | Get schema info |
| `mssql_schemas` | List schemas |
| `mssql_schema_permissions` | Get schema permissions |
| `mssql_server_role` | Get server role info |
| `mssql_server_roles` | List server roles |
| `mssql_server_permissions` | Get server permissions |
| `mssql_azuread_user` | Get Azure AD user info |
| `mssql_azuread_service_principal` | Get Azure AD SP info |
| `mssql_query` | Execute custom query |

## Quick Start

```hcl
# Create a database
resource "mssql_database" "example" {
  name = "my_database"
}

# Create a login
resource "mssql_sql_login" "example" {
  name     = "my_login"
  password = "SecurePassword123!"
}

# Create a user
resource "mssql_sql_user" "example" {
  database_name = mssql_database.example.name
  name          = "my_user"
  login_name    = mssql_sql_login.example.name
}

# Grant permissions
resource "mssql_database_permission" "example" {
  database_name  = mssql_database.example.name
  principal_name = mssql_sql_user.example.name
  permission     = "SELECT"
}
```

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/muecahit94/mssql/latest/docs) or in the [docs/](docs/) folder.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT - see [LICENSE](LICENSE)
