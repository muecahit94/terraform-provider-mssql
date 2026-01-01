#!/usr/bin/env bash
# E2E Test Suite for Azure AD Example
# This script deploys infrastructure, validates MSSQL resources, and cleans up.
# Requires: bash 4.0+, terraform, go, az-cli (logged in), sqlcmd or mssql-cli

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
INFRA_DIR="$PROJECT_ROOT/examples/testing/azure_ad/infrastructure"
RESOURCES_DIR="$PROJECT_ROOT/examples/testing/azure_ad/mssql_resources"
PROVIDER_NAME="muecahit94/mssql"

# Test results tracking
TEST_NAMES=()
TEST_STATUSES=()
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_record() {
    local name="$1"
    local status="$2"
    TEST_NAMES+=("$name")
    TEST_STATUSES+=("$status")
    ((TOTAL_TESTS++))
    if [ "$status" == "PASS" ]; then
        ((PASSED_TESTS++))
        log_success "$name"
    else
        ((FAILED_TESTS++))
        log_error "$name"
    fi
}

setup_local_provider() {
    log_info "Setting up local provider override..."
    export TF_CLI_CONFIG_FILE="$PROJECT_ROOT/.terraformrc.test"
    cat > "$TF_CLI_CONFIG_FILE" <<EOF
provider_installation {
  dev_overrides {
    "$PROVIDER_NAME" = "$PROJECT_ROOT"
  }
  direct {}
}
EOF
    log_info "Using local provider from: $PROJECT_ROOT"
}

cleanup_provider_config() {
    rm -f "$PROJECT_ROOT/.terraformrc.test"
}

cleanup_state() {
    log_info "Cleaning up state files..."
    rm -rf "$INFRA_DIR/.terraform" "$INFRA_DIR/terraform.tfstate"* "$INFRA_DIR/.terraform.lock.hcl"
    rm -rf "$RESOURCES_DIR/.terraform" "$RESOURCES_DIR/terraform.tfstate"* "$RESOURCES_DIR/.terraform.lock.hcl"
}

run_sql() {
    local host="$1"
    local user="$2"
    local password="$3"
    local db="$4"
    local query="$5"

    # Try mssql-cli (or mssql) first, then sqlcmd
    if command -v mssql-cli &> /dev/null; then
        mssql-cli -S "$host" -U "$user" -P "$password" -d "$db" -Q "$query"
    elif command -v mssql &> /dev/null; then
        mssql -S "$host" -U "$user" -P "$password" -d "$db" -Q "$query"
    elif command -v sqlcmd &> /dev/null; then
        # mssql-tools or mssql-tools18
        # -C TrustServerCertificate
        sqlcmd -S "$host" -U "$user" -P "$password" -d "$db" -Q "$query" -C
    else
        echo "Error: Neither mssql-cli, mssql, nor sqlcmd found" >&2
        return 1
    fi
}


phase_summary() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE} TEST SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
    echo -e "Passed:      ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed:      ${RED}$FAILED_TESTS${NC}"
    echo ""

    echo "Detailed Results:"
    echo "-----------------"
    for i in "${!TEST_NAMES[@]}"; do
        if [ "${TEST_STATUSES[$i]}" == "PASS" ]; then
            echo -e "  ${GREEN}✓${NC} ${TEST_NAMES[$i]}"
        else
            echo -e "  ${RED}✗${NC} ${TEST_NAMES[$i]}"
        fi
    done
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}ALL TESTS PASSED!${NC}"
        return 0
    else
        echo -e "${RED}SOME TESTS FAILED!${NC}"
        return 1
    fi
}

# Main execution
main() {
    log_info "Starting Azure AD E2E Test Sequence"

    # specific: verify az login
    if ! az account show >/dev/null 2>&1; then
        log_error "Please run 'az login' before running this script."
        exit 1
    fi

    # Auto-detect Subscription and Tenant ID
    AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
    AZURE_TENANT_ID=$(az account show --query tenantId -o tsv)

    log_info "Detected Azure Context:"
    echo "  Subscription ID: $AZURE_SUBSCRIPTION_ID"
    echo "  Tenant ID:       $AZURE_TENANT_ID"

    # Trap for cleanup
    trap 'cleanup_provider_config' EXIT

    # Phase 1: Build Provider
    log_info "Building Terraform provider..."
    cd "$PROJECT_ROOT"
    if go build -o terraform-provider-mssql .; then
         log_record "Provider Build" "PASS"
    else
         log_record "Provider Build" "FAIL"
         exit 1
    fi
    setup_local_provider
    log_record "Local Provider Setup" "PASS"

    # Phase 2: Deploy Infrastructure
    log_info "Deploying Infrastructure..."
    cd "$INFRA_DIR"
    terraform init >/dev/null

    if terraform apply -auto-approve -var="subscription_id=$AZURE_SUBSCRIPTION_ID"; then
        log_record "Infrastructure Deployment" "PASS"
    else
        log_record "Infrastructure Deployment" "FAIL"
        exit 1
    fi

    # Capture outputs
    SQL_HOST=$(terraform output -raw sql_server_fqdn)
    DB_NAME=$(terraform output -raw database_name)
    MI_NAME=$(terraform output -raw mi_name)
    MI_OBJECT_ID=$(terraform output -raw mi_principal_id)
    SQL_USER=$(terraform output -raw sql_admin_username)
    SQL_PASSWORD=$(terraform output -raw sql_admin_password)

    log_info "Infrastructure Details:"
    echo "  Host: $SQL_HOST"
    echo "  DB:   $DB_NAME"
    echo "  MI:   $MI_NAME ($MI_OBJECT_ID)"

    # Delay for Firewall Rule Propagation
    log_info "Waiting 30 seconds for Firewall Rule propagation..."
    sleep 30

    # Phase 3: Deploy MSSQL Resources
    log_info "Deploying MSSQL Resources..."
    cd "$RESOURCES_DIR"
    terraform init >/dev/null

    # We pass azure_tenant_id as it is a variable in mssql_resources usually.
    # If not, it will just warn about unused var.
    if terraform apply -auto-approve \
        -var="sql_hostname=$SQL_HOST" \
        -var="database_name=$DB_NAME" \
        -var="mi_name=$MI_NAME" \
        -var="mi_object_id=$MI_OBJECT_ID" \
        -var="azure_tenant_id=$AZURE_TENANT_ID"; then
        log_record "MSSQL Resources Deployment" "PASS"
    else
        log_record "MSSQL Resources Deployment" "FAIL"
        # We continue to verification/cleanup even if apply failed to verify state/debug
    fi

    # Verification (Phase 4)
    log_info "Verifying resources via SQL..."

    # Check MI User creation
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" "SELECT 1 FROM sys.database_principals WHERE name = '$MI_NAME'" | grep -q "1"; then
        log_record "SQL Verify: Managed Identity User Exists" "PASS"
    else
        log_record "SQL Verify: Managed Identity User Exists" "FAIL"
    fi

    # Check Role creation
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" "SELECT 1 FROM sys.database_principals WHERE name = 'managed_identity_role' AND type = 'R'" | grep -q "1"; then
        log_record "SQL Verify: Role Exists" "PASS"
    else
         log_record "SQL Verify: Role Exists" "FAIL"
    fi

    # Check Permission creation (SELECT on Database)
    # Checking if role has SELECT permission
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" "SELECT 1 FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id WHERE pr.name = 'managed_identity_role' AND p.permission_name = 'SELECT'" | grep -q "1"; then
        log_record "SQL Verify: Permission Granted" "PASS"
    else
        log_record "SQL Verify: Permission Granted" "FAIL"
    fi

    # Check Role Assignment (MI in Role)
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'managed_identity_role' AND m.name = '$MI_NAME'" | grep -q "1"; then
        log_record "SQL Verify: Role Assignment" "PASS"
    else
        log_record "SQL Verify: Role Assignment" "FAIL"
    fi

    # Phase 5: Cleanup
    log_info "Cleaning Up..."

    log_info "Destroying MSSQL Resources..."
    cd "$RESOURCES_DIR"
    if terraform destroy -auto-approve \
        -var="sql_hostname=$SQL_HOST" \
        -var="database_name=$DB_NAME" \
        -var="mi_name=$MI_NAME" \
        -var="mi_object_id=$MI_OBJECT_ID" \
        -var="azure_tenant_id=$AZURE_TENANT_ID"; then
         log_record "MSSQL Resources Destruction" "PASS"
    else
         log_record "MSSQL Resources Destruction" "FAIL"
    fi

    log_info "Destroying Infrastructure..."
    cd "$INFRA_DIR"
    if terraform destroy -auto-approve -var="subscription_id=$AZURE_SUBSCRIPTION_ID"; then
        log_record "Infrastructure Destruction" "PASS"
    else
        log_record "Infrastructure Destruction" "FAIL"
    fi

    phase_summary
}

main "$@"
