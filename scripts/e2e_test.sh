#!/usr/bin/env bash
# E2E Test Suite for MSSQL Terraform Provider
# This script runs comprehensive end-to-end tests
# Requires: bash 4.0+, docker, terraform, mssql-cli, go

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking (using indexed arrays for compatibility)
TEST_NAMES=()
TEST_STATUSES=()
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SA_PASSWORD="P@ssw0rd123!"
SQL_HOST="localhost"
SQL_PORT="1433"
PROVIDER_NAME="muecahit94/mssql"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

record_test() {
    local test_name="$1"
    local result="$2"
    TEST_NAMES+=("$test_name")
    TEST_STATUSES+=("$result")
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [[ "$result" == "PASS" ]]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "$test_name"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "$test_name"
    fi
}

run_sql() {
    local query="$1"
    local database="${2:-master}"
    mssql -u sa -p "$SA_PASSWORD" -d "$database" -q "$query" 2>/dev/null
}

wait_for_sql() {
    log_info "Waiting for SQL Server to be ready (this may take up to 60 seconds)..."
    local max_attempts=60
    local attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if run_sql "SELECT 1" >/dev/null 2>&1; then
            log_info "SQL Server responding, waiting additional 10 seconds for stability..."
            sleep 10
            # Verify again after waiting
            if run_sql "SELECT 1" >/dev/null 2>&1; then
                log_success "SQL Server is ready and stable"
                return 0
            fi
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    echo ""
    log_error "SQL Server failed to start within timeout"
    return 1
}

cleanup_state_files() {
    local dir="$1"
    log_info "Cleaning up state files in $dir"
    rm -f "$dir/terraform.tfstate" "$dir/terraform.tfstate.backup" "$dir/.terraform.lock.hcl"
    rm -rf "$dir/.terraform"
}

setup_local_provider() {
    log_info "Setting up local provider override..."

    # Create a temporary .terraformrc for this test session
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

# Phase 1: Setup
phase_setup() {
    log_header "PHASE 1: SETUP"

    cd "$PROJECT_ROOT"

    # Start Docker
    log_info "Starting Docker Compose..."
    if docker compose up -d; then
        record_test "Docker Compose Start" "PASS"
    else
        record_test "Docker Compose Start" "FAIL"
        return 1
    fi

    # Build provider
    log_info "Building Terraform provider..."
    if go build -o terraform-provider-mssql . 2>&1; then
        record_test "Provider Build" "PASS"
    else
        record_test "Provider Build" "FAIL"
        return 1
    fi

    # Setup local provider override
    setup_local_provider
    record_test "Local Provider Setup" "PASS"

    # Wait for SQL Server
    if wait_for_sql; then
        record_test "SQL Server Ready" "PASS"
    else
        record_test "SQL Server Ready" "FAIL"
        return 1
    fi

    return 0
}

# Phase 2: Complete Example
phase_complete_example() {
    log_header "PHASE 2: COMPLETE EXAMPLE"

    local example_dir="$PROJECT_ROOT/examples/testing/complete"
    cd "$example_dir"

    # Clean up any existing state
    cleanup_state_files "$example_dir"

    # Create tfvars
    cat > terraform.tfvars <<EOF
sql_hostname = "localhost"
sql_port     = 1433
sql_username = "sa"
sql_password = "$SA_PASSWORD"
app_password = "AppP@ssw0rd123!"
EOF

    log_info "Applying Terraform..."
    local apply_output
    apply_output=$(terraform apply -auto-approve 2>&1)
    echo "$apply_output" | tail -10
    if echo "$apply_output" | grep -q "Apply complete"; then
        record_test "Complete Example: terraform apply" "PASS"
    else
        record_test "Complete Example: terraform apply" "FAIL"
        log_warning "Apply output: $apply_output"
        return 1
    fi

    # SQL Verifications
    log_info "Verifying resources via SQL..."

    # Check database
    if run_sql "SELECT 1 FROM sys.databases WHERE name = 'application_db'" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: Database exists" "PASS"
    else
        record_test "SQL Verify: Database exists" "FAIL"
    fi

    # Check login
    if run_sql "SELECT 1 FROM sys.sql_logins WHERE name = 'app_login'" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: Login exists" "PASS"
    else
        record_test "SQL Verify: Login exists" "FAIL"
    fi

    # Check user in database
    if run_sql "SELECT 1 FROM sys.database_principals WHERE name = 'app_user'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: User exists in database" "PASS"
    else
        record_test "SQL Verify: User exists in database" "FAIL"
    fi

    # Check role
    if run_sql "SELECT 1 FROM sys.database_principals WHERE name = 'app_readers' AND type = 'R'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: Role exists" "PASS"
    else
        record_test "SQL Verify: Role exists" "FAIL"
    fi

    # Check schema
    if run_sql "SELECT 1 FROM sys.schemas WHERE name = 'app'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: Schema exists" "PASS"
    else
        record_test "SQL Verify: Schema exists" "FAIL"
    fi

    # Check role membership (app_user in app_readers via OPTION 1: inline roles)
    if run_sql "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'app_readers' AND m.name = 'app_user'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: app_user in app_readers (Option 1: inline roles)" "PASS"
    else
        record_test "SQL Verify: app_user in app_readers (Option 1: inline roles)" "FAIL"
    fi

    # Check writers role exists
    if run_sql "SELECT 1 FROM sys.database_principals WHERE name = 'app_writers' AND type = 'R'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: app_writers role exists" "PASS"
    else
        record_test "SQL Verify: app_writers role exists" "FAIL"
    fi

    # Check test_user in app_readers (via OPTION 2: explicit role_member)
    if run_sql "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'app_readers' AND m.name = 'test_user'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: test_user in app_readers (Option 2: role_member)" "PASS"
    else
        record_test "SQL Verify: test_user in app_readers (Option 2: role_member)" "FAIL"
    fi

    # Check test_user in app_writers (via OPTION 2: explicit role_member)
    if run_sql "SELECT 1 FROM sys.database_role_members rm JOIN sys.database_principals r ON rm.role_principal_id = r.principal_id JOIN sys.database_principals m ON rm.member_principal_id = m.principal_id WHERE r.name = 'app_writers' AND m.name = 'test_user'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: test_user in app_writers (Option 2: role_member)" "PASS"
    else
        record_test "SQL Verify: test_user in app_writers (Option 2: role_member)" "FAIL"
    fi

    # Check permission
    if run_sql "SELECT 1 FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id WHERE pr.name = 'app_readers' AND p.permission_name = 'SELECT'" "application_db" | grep -v "Executed in" | grep "1" -q; then
        record_test "SQL Verify: Permission granted" "PASS"
    else
        record_test "SQL Verify: Permission granted" "FAIL"
    fi

    # Check test_user has SELECT on app schema with WITH GRANT OPTION (state = W)
    local test_user_perm=$(run_sql "SELECT state FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id JOIN sys.schemas s ON p.major_id = s.schema_id WHERE pr.name = 'test_user' AND s.name = 'app' AND p.permission_name = 'SELECT'" "application_db" 2>/dev/null)
    if echo "$test_user_perm" | grep -q "W"; then
        record_test "SQL Verify: test_user schema permission" "PASS"
    else
        log_error "Expected test_user to have SELECT WITH GRANT OPTION (W)"
        record_test "SQL Verify: test_user schema permission" "FAIL"
    fi

    # Check app_user owns the app schema
    local app_schema_owner=$(run_sql "SELECT dp.name FROM sys.schemas s JOIN sys.database_principals dp ON s.principal_id = dp.principal_id WHERE s.name = 'app'" "application_db" 2>/dev/null)
    if echo "$app_schema_owner" | grep -q "app_user"; then
        record_test "SQL Verify: app_user schema owner" "PASS"
    else
        log_error "Expected app_user to own the app schema"
        record_test "SQL Verify: app_user schema owner" "FAIL"
    fi

    # Check idempotency
    log_info "Checking idempotency..."
    local plan_output
    plan_output=$(terraform plan -detailed-exitcode 2>&1) || true
    if echo "$plan_output" | grep -q "No changes"; then
        record_test "Complete Example: Idempotency" "PASS"
    else
        record_test "Complete Example: Idempotency" "FAIL"
    fi

    return 0
}

# Phase 3: Data Sources
phase_data_sources() {
    log_header "PHASE 3: DATA SOURCES EXAMPLE"

    local example_dir="$PROJECT_ROOT/examples/testing/data_sources"
    cd "$example_dir"

    # Clean up any existing state
    cleanup_state_files "$example_dir"

    log_info "Applying Terraform..."
    local apply_output
    apply_output=$(terraform apply -auto-approve 2>&1)
    echo "$apply_output" | tail -10
    if echo "$apply_output" | grep -q "Apply complete"; then
        record_test "Data Sources: terraform apply" "PASS"
    else
        record_test "Data Sources: terraform apply" "FAIL"
        return 1
    fi

    # Verify outputs work
    log_info "Verifying data source outputs..."
    if terraform output 2>&1 | grep -q "databases"; then
        record_test "Data Sources: Output verification" "PASS"
    else
        record_test "Data Sources: Output verification" "FAIL"
    fi

    return 0
}

# Phase 4: Provider Example
phase_provider_example() {
    log_header "PHASE 4: PROVIDER EXAMPLE"

    local example_dir="$PROJECT_ROOT/examples/testing/provider"
    cd "$example_dir"

    # Clean up any existing state
    cleanup_state_files "$example_dir"

    log_info "Applying Terraform..."
    local apply_output
    apply_output=$(terraform apply -auto-approve 2>&1)
    echo "$apply_output" | tail -10
    if echo "$apply_output" | grep -q "Apply complete"; then
        record_test "Provider Example: terraform apply" "PASS"
    else
        record_test "Provider Example: terraform apply" "FAIL"
        return 1
    fi

    # Verify it worked by checking resources were created
    log_info "Verifying provider example resources..."
    if run_sql "SELECT 1 FROM sys.databases WHERE name = 'example_db'" | grep -v "Executed in" | grep "1" -q; then
        record_test "Provider Example: Resources verified" "PASS"
    else
        record_test "Provider Example: Resources verified" "FAIL"
    fi

    # Destroy
    log_info "Destroying provider example..."
    local destroy_output
    destroy_output=$(terraform destroy -auto-approve 2>&1)
    echo "$destroy_output" | tail -5
    if echo "$destroy_output" | grep -q "Destroy complete"; then
        record_test "Provider Example: terraform destroy" "PASS"
    else
        record_test "Provider Example: terraform destroy" "FAIL"
    fi

    return 0
}

# Phase 5: Drift Recovery
phase_drift_recovery() {
    log_header "PHASE 5: DRIFT RECOVERY TESTS"

    local example_dir="$PROJECT_ROOT/examples/testing/complete"
    cd "$example_dir"

    # Test 1: Delete role and recover
    log_info "Test: Role deletion recovery..."
    run_sql "ALTER ROLE app_readers DROP MEMBER app_user; DROP ROLE app_readers;" "application_db" >/dev/null 2>&1 || true

    local apply_output
    apply_output=$(terraform apply -auto-approve 2>&1)
    if echo "$apply_output" | grep -q "Apply complete"; then
        # Verify role was recreated
        if run_sql "SELECT 1 FROM sys.database_principals WHERE name = 'app_readers'" "application_db" | grep -v "Executed in" | grep "1" -q; then
            record_test "Drift Recovery: Role recreation" "PASS"
        else
            record_test "Drift Recovery: Role recreation" "FAIL"
        fi
    else
        record_test "Drift Recovery: Role recreation" "FAIL"
    fi

    # Test 2: Delete user and recover
    log_info "Test: User deletion recovery..."
    run_sql "DROP SCHEMA app; DROP USER app_user;" "application_db" >/dev/null 2>&1 || true

    apply_output=$(terraform apply -auto-approve 2>&1)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "SELECT 1 FROM sys.database_principals WHERE name = 'app_user'" "application_db" | grep -v "Executed in" | grep "1" -q; then
            record_test "Drift Recovery: User recreation" "PASS"
        else
            record_test "Drift Recovery: User recreation" "FAIL"
        fi
    else
        record_test "Drift Recovery: User recreation" "FAIL"
    fi

    # Test 3: Revoke permission and recover
    log_info "Test: Permission recovery..."
    run_sql "REVOKE SELECT FROM app_readers" "application_db" >/dev/null 2>&1 || true

    apply_output=$(terraform apply -auto-approve 2>&1)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "SELECT 1 FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id WHERE pr.name = 'app_readers' AND p.permission_name = 'SELECT'" "application_db" | grep -v "Executed in" | grep "1" -q; then
            record_test "Drift Recovery: Permission restoration" "PASS"
        else
            record_test "Drift Recovery: Permission restoration" "FAIL"
        fi
    else
        record_test "Drift Recovery: Permission restoration" "FAIL"
    fi

    # Test 4: Disable login and recover
    log_info "Test: Login modification recovery..."
    run_sql "ALTER LOGIN app_login DISABLE" >/dev/null 2>&1 || true

    apply_output=$(terraform apply -auto-approve 2>&1)
    if echo "$apply_output" | grep -q "Apply complete"; then
        if run_sql "SELECT 1 FROM sys.sql_logins WHERE name = 'app_login' AND is_disabled = 0" | grep -v "Executed in" | grep "1" -q; then
            record_test "Drift Recovery: Login re-enable" "PASS"
        else
            record_test "Drift Recovery: Login re-enable" "FAIL"
        fi
    else
        echo "Terraform apply failed:"
        echo "$apply_output"
        record_test "Drift Recovery: Login re-enable" "FAIL"
    fi

    # Test 5: Schema permission drift recovery
    log_info "Test: Schema permission drift recovery..."

    # Revoke the test_user's SELECT permission on the app schema
    run_sql "REVOKE SELECT ON SCHEMA::app FROM test_user CASCADE" "application_db" >/dev/null 2>&1 || true

    apply_output=$(terraform apply -auto-approve 2>&1)
    if echo "$apply_output" | grep -q "Apply complete"; then
        # Verify the permission was restored with WITH GRANT OPTION
        local restored_perm=$(run_sql "SELECT state FROM sys.database_permissions p JOIN sys.database_principals pr ON p.grantee_principal_id = pr.principal_id JOIN sys.schemas s ON p.major_id = s.schema_id WHERE pr.name = 'test_user' AND s.name = 'app' AND p.permission_name = 'SELECT'" "application_db" 2>/dev/null)
        if echo "$restored_perm" | grep -q "W"; then
            record_test "Drift Recovery: Schema permission restoration" "PASS"
        else
            record_test "Drift Recovery: Schema permission restoration" "FAIL"
        fi
    else
        record_test "Drift Recovery: Schema permission restoration" "FAIL"
    fi

    return 0
}


# Phase 6: Cleanup
phase_cleanup() {
    log_header "PHASE 6: CLEANUP"

    # Destroy complete example
    log_info "Destroying complete example..."
    cd "$PROJECT_ROOT/examples/testing/complete"
    terraform destroy -auto-approve 2>&1 | tail -3 || true

    # Stop Docker
    log_info "Stopping Docker Compose..."
    cd "$PROJECT_ROOT"
    docker compose down 2>&1 || true

    # Clean up all state files
    log_info "Cleaning up state files..."
    cleanup_state_files "$PROJECT_ROOT/examples/testing/complete"
    cleanup_state_files "$PROJECT_ROOT/examples/testing/data_sources"
    cleanup_state_files "$PROJECT_ROOT/examples/testing/provider"

    # Clean up tfvars
    rm -f "$PROJECT_ROOT/examples/testing/complete/terraform.tfvars"
    rm -f "$PROJECT_ROOT/examples/testing/data_sources/terraform.tfvars"

    # Clean up provider config
    cleanup_provider_config

    record_test "Cleanup" "PASS"

    return 0
}

# Phase 7: Summary
phase_summary() {
    log_header "TEST SUMMARY"

    echo ""
    echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
    echo -e "Passed:      ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed:      ${RED}$FAILED_TESTS${NC}"
    echo ""

    echo "Detailed Results:"
    echo "-----------------"
    local i=0
    while [[ $i -lt ${#TEST_NAMES[@]} ]]; do
        local name="${TEST_NAMES[$i]}"
        local status="${TEST_STATUSES[$i]}"
        if [[ "$status" == "PASS" ]]; then
            echo -e "  ${GREEN}✓${NC} $name"
        else
            echo -e "  ${RED}✗${NC} $name"
        fi
        i=$((i + 1))
    done
    echo ""

    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN} ALL TESTS PASSED!${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        return 0
    else
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${RED} SOME TESTS FAILED!${NC}"
        echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
        return 1
    fi
}

# Main execution
main() {
    log_header "MSSQL TERRAFORM PROVIDER E2E TEST SUITE"
    echo "Started at: $(date)"
    echo ""

    # Trap to ensure cleanup on exit
    trap 'cleanup_provider_config' EXIT

    # Run all phases
    phase_setup || { phase_cleanup; phase_summary; exit 1; }
    phase_complete_example || true
    phase_data_sources || true
    phase_provider_example || true
    phase_drift_recovery || true
    phase_cleanup

    # Print summary
    phase_summary
    exit $?
}

# Run main
main "$@"
