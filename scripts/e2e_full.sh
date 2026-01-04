#!/usr/bin/env bash
# Combined E2E Test Runner
# Runs both local and Azure e2e tests and provides a combined summary

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Result storage
LOCAL_RESULTS_FILE=$(mktemp)
AZURE_RESULTS_FILE=$(mktemp)
LOCAL_EXIT=0
AZURE_EXIT=0

cleanup() {
    rm -f "$LOCAL_RESULTS_FILE" "$AZURE_RESULTS_FILE"
}
trap cleanup EXIT

echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           COMBINED E2E TEST SUITE                             ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# ============================================================================
# Run Local E2E Tests
# ============================================================================
echo -e "${BLUE}┌───────────────────────────────────────────────────────────────┐${NC}"
echo -e "${BLUE}│ RUNNING: Local E2E Tests (Docker)                             │${NC}"
echo -e "${BLUE}└───────────────────────────────────────────────────────────────┘${NC}"
echo ""

if "$SCRIPT_DIR/e2e_test.sh" 2>&1 | tee "$LOCAL_RESULTS_FILE"; then
    LOCAL_EXIT=0
else
    LOCAL_EXIT=${PIPESTATUS[0]}
fi

echo ""

# ============================================================================
# Run Azure E2E Tests
# ============================================================================
echo -e "${BLUE}┌───────────────────────────────────────────────────────────────┐${NC}"
echo -e "${BLUE}│ RUNNING: Azure AD E2E Tests                                   │${NC}"
echo -e "${BLUE}└───────────────────────────────────────────────────────────────┘${NC}"
echo ""

if "$SCRIPT_DIR/e2e_test_azure.sh" 2>&1 | tee "$AZURE_RESULTS_FILE"; then
    AZURE_EXIT=0
else
    AZURE_EXIT=${PIPESTATUS[0]}
fi

# ============================================================================
# Parse and Combine Results
# ============================================================================
echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           COMBINED TEST SUMMARY                               ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Helper function to strip ANSI codes
strip_ansi() {
    sed 's/\x1b\[[0-9;]*m//g' | tr -d '\r'
}

# Extract test counts from local results (strip ANSI codes first)
LOCAL_TOTAL=$(cat "$LOCAL_RESULTS_FILE" | strip_ansi | grep -E "^Total Tests:" | tail -1 | awk '{print $NF}')
LOCAL_PASSED=$(cat "$LOCAL_RESULTS_FILE" | strip_ansi | grep -E "^Passed:" | tail -1 | awk '{print $NF}')
LOCAL_FAILED=$(cat "$LOCAL_RESULTS_FILE" | strip_ansi | grep -E "^Failed:" | tail -1 | awk '{print $NF}')

# Extract test counts from azure results (strip ANSI codes first)
AZURE_TOTAL=$(cat "$AZURE_RESULTS_FILE" | strip_ansi | grep -E "^Total Tests:" | tail -1 | awk '{print $NF}')
AZURE_PASSED=$(cat "$AZURE_RESULTS_FILE" | strip_ansi | grep -E "^Passed:" | tail -1 | awk '{print $NF}')
AZURE_FAILED=$(cat "$AZURE_RESULTS_FILE" | strip_ansi | grep -E "^Failed:" | tail -1 | awk '{print $NF}')

# Ensure numeric values (default to 0 if empty or non-numeric)
LOCAL_TOTAL=$(echo "${LOCAL_TOTAL:-0}" | grep -oE '^[0-9]+$' || echo "0")
LOCAL_PASSED=$(echo "${LOCAL_PASSED:-0}" | grep -oE '^[0-9]+$' || echo "0")
LOCAL_FAILED=$(echo "${LOCAL_FAILED:-0}" | grep -oE '^[0-9]+$' || echo "0")
AZURE_TOTAL=$(echo "${AZURE_TOTAL:-0}" | grep -oE '^[0-9]+$' || echo "0")
AZURE_PASSED=$(echo "${AZURE_PASSED:-0}" | grep -oE '^[0-9]+$' || echo "0")
AZURE_FAILED=$(echo "${AZURE_FAILED:-0}" | grep -oE '^[0-9]+$' || echo "0")

# Calculate combined totals
COMBINED_TOTAL=$((LOCAL_TOTAL + AZURE_TOTAL))
COMBINED_PASSED=$((LOCAL_PASSED + AZURE_PASSED))
COMBINED_FAILED=$((LOCAL_FAILED + AZURE_FAILED))

# Display results by suite
echo -e "${BLUE}Local E2E Tests:${NC}"
if [ "$LOCAL_EXIT" -eq 0 ]; then
    echo -e "  Status:  ${GREEN}PASSED${NC}"
else
    echo -e "  Status:  ${RED}FAILED${NC}"
fi
echo -e "  Tests:   ${LOCAL_TOTAL} total, ${GREEN}${LOCAL_PASSED} passed${NC}, ${RED}${LOCAL_FAILED} failed${NC}"
echo ""

echo -e "${BLUE}Azure AD E2E Tests:${NC}"
if [ "$AZURE_EXIT" -eq 0 ]; then
    echo -e "  Status:  ${GREEN}PASSED${NC}"
else
    echo -e "  Status:  ${RED}FAILED${NC}"
fi
echo -e "  Tests:   ${AZURE_TOTAL} total, ${GREEN}${AZURE_PASSED} passed${NC}, ${RED}${AZURE_FAILED} failed${NC}"
echo ""

echo -e "${BLUE}───────────────────────────────────────────────────────────────${NC}"
echo ""
echo -e "${BLUE}Combined Results:${NC}"
echo -e "  Total Tests:  ${BLUE}${COMBINED_TOTAL}${NC}"
echo -e "  Passed:       ${GREEN}${COMBINED_PASSED}${NC}"
echo -e "  Failed:       ${RED}${COMBINED_FAILED}${NC}"
echo ""

# Extract and display all test results (keep colors for detailed output)
echo "Detailed Results:"
echo "-----------------"
echo ""
echo -e "${YELLOW}[Local Tests]${NC}"
# Match lines with checkmark/cross/circle - they may have color codes before the symbol
grep -E "✓|✗|○" "$LOCAL_RESULTS_FILE" | grep -v "^=" | head -40 || echo "  (no results found)"
echo ""
echo -e "${YELLOW}[Azure Tests]${NC}"
grep -E "✓|✗|○" "$AZURE_RESULTS_FILE" | grep -v "^=" | head -40 || echo "  (no results found)"
echo ""

# Final verdict
if [ "$COMBINED_FAILED" -eq 0 ] && [ "$LOCAL_EXIT" -eq 0 ] && [ "$AZURE_EXIT" -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           ALL E2E TESTS PASSED!                               ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║           SOME E2E TESTS FAILED!                              ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}"
    exit 1
fi
