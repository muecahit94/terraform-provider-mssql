# Azure AD Testing Example

This example demonstrates testing the mssql provider with Azure SQL and Azure AD authentication.

## Structure

This example is split into two separate Terraform configurations:

```
azure_ad/
├── infrastructure/     # Phase 1: Azure SQL Server & Database
│   ├── providers.tf
│   ├── variables.tf
│   ├── main.tf
│   └── outputs.tf
└── mssql_resources/    # Phase 2: MSSQL Provider Resources
    ├── providers.tf
    ├── variables.tf
    ├── main.tf
    └── outputs.tf
```

**Why two configurations?**
The mssql provider needs to connect to SQL Server during `terraform plan`. Since the SQL Server doesn't exist yet in Phase 1, we need separate configurations.

## Quick Start

### Prerequisites

1. Azure CLI logged in: `az login`
2. Terraform installed

### Phase 1: Create Azure Infrastructure

```bash
cd infrastructure

# Create terraform.tfvars
cat > terraform.tfvars << EOF
sql_server_name    = "sql-mssql-test-UNIQUE"  # Must be globally unique
sql_admin_password = "YourStrongP@ssw0rd123!"
EOF

terraform init
terraform apply
```

Note the `sql_server_fqdn` output - you'll need it for Phase 2.

### Phase 2: Create MSSQL Resources

```bash
cd ../mssql_resources

# Create terraform.tfvars using outputs from Phase 1
cat > terraform.tfvars << EOF
sql_hostname  = "sql-mssql-test-UNIQUE.database.windows.net"  # From Phase 1 output
database_name = "testdb"

# Optional: Create Azure AD user
developer_email     = "developer@yourdomain.com"
developer_object_id = "azure-ad-object-id"

# Optional: Create Azure AD service principal
app_name      = "my-application"
app_client_id = "app-client-id"
EOF

terraform init
terraform apply
```

## Authentication

The mssql provider uses **Azure AD authentication** with the default credential chain:

1. Environment variables (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`)
2. Workload Identity (Kubernetes)
3. Managed Identity (Azure VMs/App Services)
4. **Azure CLI** (`az login`) ← Most common for local development
5. Azure PowerShell
6. Azure Developer CLI

For local development, just run `az login` before applying.

## Clean Up

```bash
# Destroy mssql resources first
cd mssql_resources
terraform destroy

# Then destroy infrastructure
cd ../infrastructure
terraform destroy
```
