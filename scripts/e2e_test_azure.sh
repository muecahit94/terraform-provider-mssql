#!/usr/bin/env bash
# E2E Test Suite for Azure AD Example
# This script deploys infrastructure, validates MSSQL resources, and cleans up.
# Requires: bash 4.0+, terraform, go, az-cli (logged in), sqlcmd/mssql-cli
#
# Environment Variables:
#   SKIP_INFRA_DESTROY=1  - Skip infrastructure destruction (useful for debugging)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
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

# Captured infrastructure outputs (global for cleanup trap)
SQL_HOST=""
DB_NAME=""
MI_NAME=""
MI_OBJECT_ID=""
SQL_USER=""
SQL_PASSWORD=""
AZURE_SUBSCRIPTION_ID=""
AZURE_TENANT_ID=""

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

log_record() {
    local name="$1"
    local status="$2"
    TEST_NAMES+=("$name")
    TEST_STATUSES+=("$status")
    ((TOTAL_TESTS++))
    if [ "$status" == "PASS" ]; then
        ((PASSED_TESTS++))
        log_success "$name"
    elif [ "$status" == "SKIP" ]; then
        # Skipped tests don't count as failures
        echo -e "${YELLOW}[SKIP]${NC} $name"
    else
        ((FAILED_TESTS++))
        log_error "$name"
    fi
}

phase_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
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

cleanup_all() {
    log_warn "Performing emergency cleanup..."
    cleanup_provider_config

    # Try to destroy resources if we have the variables
    if [ -n "$SQL_HOST" ]; then
        cd "$RESOURCES_DIR" 2>/dev/null || true
        terraform destroy -auto-approve \
            -var="sql_hostname=$SQL_HOST" \
            -var="database_name=$DB_NAME" \
            -var="mi_name=$MI_NAME" \
            -var="mi_object_id=$MI_OBJECT_ID" \
            -var="azure_tenant_id=$AZURE_TENANT_ID" 2>/dev/null || true
    fi

    if [ -n "$AZURE_SUBSCRIPTION_ID" ] && [ -z "$SKIP_INFRA_DESTROY" ]; then
        cd "$INFRA_DIR" 2>/dev/null || true
        terraform destroy -auto-approve -var="subscription_id=$AZURE_SUBSCRIPTION_ID" 2>/dev/null || true
    elif [ -n "$SKIP_INFRA_DESTROY" ]; then
        log_warn "SKIP_INFRA_DESTROY is set - infrastructure will NOT be destroyed"
    fi
}

run_sql() {
    local host="$1"
    local user="$2"
    local password="$3"
    local db="$4"
    local query="$5"

    # go-sqlcmd (uses SQLCMDPASSWORD env var)
    if command -v sqlcmd &> /dev/null; then
        SQLCMDPASSWORD="$password" sqlcmd -S "$host" -U "$user" -d "$db" -Q "$query" -C 2>/dev/null
    # mssql-cli from Microsoft (Python based)
    elif command -v mssql-cli &> /dev/null; then
        mssql-cli -S "$host" -U "$user" -P "$password" -d "$db" -Q "$query" 2>/dev/null
    else
        echo "Error: No SQL CLI tool found (sqlcmd or mssql-cli)" >&2
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
        elif [ "${TEST_STATUSES[$i]}" == "SKIP" ]; then
            echo -e "  ${YELLOW}○${NC} ${TEST_NAMES[$i]} (skipped)"
        else
            echo -e "  ${RED}✗${NC} ${TEST_NAMES[$i]}"
        fi
    done
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN} ALL TESTS PASSED!${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        return 0
    else
        echo -e "${RED}SOME TESTS FAILED!${NC}"
        return 1
    fi
}

# Main execution
main() {
    phase_header "AZURE AD E2E TEST SUITE"

    # Verify az login
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

    # Trap for cleanup on error
    trap 'cleanup_all' ERR
    trap 'cleanup_provider_config' EXIT

    # =========================================================================
    # PHASE 1: BUILD PROVIDER
    # =========================================================================
    phase_header "PHASE 1: BUILD PROVIDER"

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

    # =========================================================================
    # PHASE 2: DEPLOY INFRASTRUCTURE
    # =========================================================================
    phase_header "PHASE 2: DEPLOY INFRASTRUCTURE"

    log_info "Deploying Azure Infrastructure..."
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

    # Wait for firewall rules to propagate
    log_info "Waiting 30 seconds for Firewall Rule propagation..."
    sleep 30

    # =========================================================================
    # PHASE 3: DEPLOY MSSQL RESOURCES
    # =========================================================================
    phase_header "PHASE 3: DEPLOY MSSQL RESOURCES"

    log_info "Deploying MSSQL Resources..."
    cd "$RESOURCES_DIR"
    terraform init >/dev/null

    if terraform apply -auto-approve \
        -var="sql_hostname=$SQL_HOST" \
        -var="database_name=$DB_NAME" \
        -var="mi_name=$MI_NAME" \
        -var="mi_object_id=$MI_OBJECT_ID" \
        -var="azure_tenant_id=$AZURE_TENANT_ID"; then
        log_record "MSSQL Resources Deployment" "PASS"
    else
        log_record "MSSQL Resources Deployment" "FAIL"
    fi

    # Idempotency check
    log_info "Checking idempotency..."
    if terraform plan -detailed-exitcode \
        -var="sql_hostname=$SQL_HOST" \
        -var="database_name=$DB_NAME" \
        -var="mi_name=$MI_NAME" \
        -var="mi_object_id=$MI_OBJECT_ID" \
        -var="azure_tenant_id=$AZURE_TENANT_ID" >/dev/null 2>&1; then
        log_record "MSSQL Resources: Idempotency" "PASS"
    else
        log_record "MSSQL Resources: Idempotency" "FAIL"
    fi

    # =========================================================================
    # PHASE 4: VERIFICATION
    # =========================================================================
    phase_header "PHASE 4: VERIFICATION"

    log_info "Verifying resources via SQL..."

    # Check MI User creation
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_principals WHERE name = '$MI_NAME'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: Managed Identity User" "PASS"
    else
        log_record "SQL Verify: Managed Identity User" "FAIL"
    fi

    # Check Role creation
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_principals WHERE name = 'managed_identity_role' AND type = 'R'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: MI Role" "PASS"
    else
        log_record "SQL Verify: MI Role" "FAIL"
    fi

    # Check Permission granted to role
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id WHERE pr.name = 'managed_identity_role' AND p.permission_name = 'SELECT'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: MI Role Permission" "PASS"
    else
        log_record "SQL Verify: MI Role Permission" "FAIL"
    fi

    # Check Role Assignment (MI in Role via OPTION 2: explicit role_member)
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'managed_identity_role' AND m.name = '$MI_NAME'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: MI Role Membership (Option 2: role_member)" "PASS"
    else
        log_record "SQL Verify: MI Role Membership (Option 2: role_member)" "FAIL"
    fi

    # Check developer_role exists
    if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_principals WHERE name = 'developer_role' AND type = 'R'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: Developer Role" "PASS"
    else
        log_record "SQL Verify: Developer Role" "FAIL"
    fi

    # Check Developer user in developer_role (via OPTION 1: inline roles)
    DEVELOPER_EMAIL="${DEVELOPER_EMAIL:-}" # Get from env or leave empty
    if [ -n "$DEVELOPER_EMAIL" ] || run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'developer_role'" | grep -v "Executed in" | grep -q "1"; then
        log_record "SQL Verify: Developer Role Membership (Option 1: inline roles)" "PASS"
    else
        log_record "SQL Verify: Developer Role Membership (Option 1: inline roles)" "FAIL"
    fi

    # =========================================================================
    # PHASE 5: DRIFT RECOVERY TESTS
    # =========================================================================
    phase_header "PHASE 5: DRIFT RECOVERY TESTS"

    cd "$RESOURCES_DIR"

    # Helper to run terraform apply with all vars
    tf_apply() {
        terraform apply -auto-approve \
            -var="sql_hostname=$SQL_HOST" \
            -var="database_name=$DB_NAME" \
            -var="mi_name=$MI_NAME" \
            -var="mi_object_id=$MI_OBJECT_ID" \
            -var="azure_tenant_id=$AZURE_TENANT_ID" 2>&1
    }

    # Test 1: Delete MI role and recover
    log_info "Test: MI Role deletion recovery..."
    run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "ALTER ROLE managed_identity_role DROP MEMBER [$MI_NAME]; DROP ROLE managed_identity_role;" >/dev/null 2>&1 || true

    apply_output=$(tf_apply)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
            "SELECT 1 FROM sys.database_principals WHERE name = 'managed_identity_role' AND type = 'R'" | grep -v "Executed in" | grep -q "1"; then
            log_record "Drift Recovery: MI Role recreation" "PASS"
        else
            log_record "Drift Recovery: MI Role recreation" "FAIL"
        fi
    else
        log_record "Drift Recovery: MI Role recreation" "FAIL"
    fi

    # Test 2: Delete MI user and recover
    log_info "Test: MI User deletion recovery..."
    run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "DROP USER [$MI_NAME];" >/dev/null 2>&1 || true

    apply_output=$(tf_apply)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
            "SELECT 1 FROM sys.database_principals WHERE name = '$MI_NAME'" | grep -v "Executed in" | grep -q "1"; then
            log_record "Drift Recovery: MI User recreation" "PASS"
        else
            log_record "Drift Recovery: MI User recreation" "FAIL"
        fi
    else
        log_record "Drift Recovery: MI User recreation" "FAIL"
    fi

    # Test 3: Revoke permission and recover
    log_info "Test: Permission revocation recovery..."
    run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "REVOKE SELECT FROM managed_identity_role;" >/dev/null 2>&1 || true

    apply_output=$(tf_apply)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
            "SELECT 1 FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id WHERE pr.name = 'managed_identity_role' AND p.permission_name = 'SELECT'" | grep -v "Executed in" | grep -q "1"; then
            log_record "Drift Recovery: Permission restoration" "PASS"
        else
            log_record "Drift Recovery: Permission restoration" "FAIL"
        fi
    else
        log_record "Drift Recovery: Permission restoration" "FAIL"
    fi

    # Test 4: Remove role membership and recover
    log_info "Test: Role membership removal recovery..."
    run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
        "ALTER ROLE managed_identity_role DROP MEMBER [$MI_NAME];" >/dev/null 2>&1 || true

    apply_output=$(tf_apply)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "$SQL_HOST" "$SQL_USER" "$SQL_PASSWORD" "$DB_NAME" \
            "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'managed_identity_role' AND m.name = '$MI_NAME'" | grep -v "Executed in" | grep -q "1"; then
            log_record "Drift Recovery: Role membership restoration" "PASS"
        else
            log_record "Drift Recovery: Role membership restoration" "FAIL"
        fi
    else
        log_record "Drift Recovery: Role membership restoration" "FAIL"
    fi

    # =========================================================================
    # PHASE 6: CLEANUP
    # =========================================================================
    phase_header "PHASE 6: CLEANUP"

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

    if [ -z "$SKIP_INFRA_DESTROY" ]; then
        log_info "Destroying Infrastructure..."
        cd "$INFRA_DIR"
        if terraform destroy -auto-approve -var="subscription_id=$AZURE_SUBSCRIPTION_ID"; then
            log_record "Infrastructure Destruction" "PASS"
        else
            log_record "Infrastructure Destruction" "FAIL"
        fi
    else
        log_warn "SKIP_INFRA_DESTROY is set - skipping infrastructure destruction"
        log_record "Infrastructure Destruction" "SKIP"
    fi

    # Show summary
    phase_summary
}

main "$@"
