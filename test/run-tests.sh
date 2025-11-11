#!/bin/bash
# Automated test runner for Exasol Terraform Provider
# Runs all test configurations and validates no drift occurs

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0
TOTAL=0

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    if [ "$status" == "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $message"
    elif [ "$status" == "FAIL" ]; then
        echo -e "${RED}✗ FAIL${NC}: $message"
    elif [ "$status" == "INFO" ]; then
        echo -e "${YELLOW}ℹ INFO${NC}: $message"
    fi
}

# Function to run a single test
run_test() {
    local test_dir=$1
    local test_name=$2

    TOTAL=$((TOTAL + 1))
    echo ""
    echo "=========================================="
    echo "Test $TOTAL: $test_name"
    echo "Directory: $test_dir"
    echo "=========================================="

    cd "$test_dir"

    # Run setup script if it exists
    if [ -f "setup.sh" ]; then
        print_status "INFO" "Running setup script..."
        if ! ./setup.sh > /dev/null 2>&1; then
            print_status "FAIL" "setup script failed"
            FAILED=$((FAILED + 1))
            cd - > /dev/null
            return 1
        fi
    fi

    # Initialize
    print_status "INFO" "Running terraform init..."
    if ! terraform init > /dev/null 2>&1; then
        print_status "FAIL" "terraform init failed"
        FAILED=$((FAILED + 1))
        cd - > /dev/null
        return 1
    fi

    # Apply
    print_status "INFO" "Running terraform apply..."
    if ! terraform apply -auto-approve > /dev/null 2>&1; then
        print_status "FAIL" "terraform apply failed"
        FAILED=$((FAILED + 1))
        cd - > /dev/null
        return 1
    fi

    # Check for drift (most important check)
    print_status "INFO" "Checking for drift..."
    local plan_output=$(terraform plan 2>&1)

    if echo "$plan_output" | grep -q "No changes"; then
        print_status "PASS" "No drift detected after apply"
        PASSED=$((PASSED + 1))
    else
        print_status "FAIL" "Drift detected after apply"
        echo "$plan_output"
        FAILED=$((FAILED + 1))
        cd - > /dev/null
        return 1
    fi

    # Destroy
    print_status "INFO" "Running terraform destroy..."
    if ! terraform destroy -auto-approve > /dev/null 2>&1; then
        print_status "FAIL" "terraform destroy failed (non-critical)"
    fi

    # Cleanup
    rm -rf .terraform terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl

    cd - > /dev/null
    return 0
}

# Main execution
echo "=========================================="
echo "Exasol Terraform Provider Test Suite"
echo "=========================================="
echo ""

# Check prerequisites
if ! command -v terraform &> /dev/null; then
    print_status "FAIL" "terraform command not found"
    exit 1
fi

if ! docker ps | grep -q exasol; then
    print_status "FAIL" "Exasol Docker container not running"
    echo "Please start Exasol container first:"
    echo "docker run -d -p 8563:8563 --name exasol exasol/docker-db:latest"
    exit 1
fi

print_status "INFO" "Prerequisites check passed"

# Get test directory base path
TEST_BASE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Define test suites (simple arrays for compatibility)
TEST_NAMES=(
    "admin-transitions"
    "suite-1-role-grants"
    "suite-2-object-privileges"
    "suite-3-system-privileges"
    "suite-4-connection-grants"
    "suite-5-real-world"
)

# Run tests
for test_name in "${TEST_NAMES[@]}"; do
    test_path="$TEST_BASE_DIR/$test_name"

    # Check if test exists
    if [ ! -e "$test_path" ]; then
        print_status "INFO" "Skipping $test_name (not implemented yet)"
        continue
    fi

    # Run test
    run_test "$test_path" "$test_name"
done

# Summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Total:  $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    print_status "PASS" "All tests passed!"
    exit 0
else
    print_status "FAIL" "$FAILED test(s) failed"
    exit 1
fi
