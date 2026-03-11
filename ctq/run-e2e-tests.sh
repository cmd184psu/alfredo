#!/bin/bash
# End-to-end test runner for CTQ

set -e

echo "=== CTQ End-to-End Tests ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

run_test() {
    local test_name=$1
    echo -n "Running $test_name... "
    if go test -v -run "$test_name" 2>&1 | grep -q "PASS"; then
        echo -e "${GREEN}✓ PASS${NC}"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        return 1
    fi
}

failed=0

# Run individual test suites
echo "Testing core workflow..."
run_test "TestEndToEnd" || ((failed++))

echo ""
echo "Testing priority scheduling..."
run_test "TestTaskPriority" || ((failed++))

echo ""
echo "Testing cooldown mechanism..."
run_test "TestTaskCooldown" || ((failed++))

echo ""
echo "Testing retry logic..."
run_test "TestTaskRetry" || ((failed++))

echo ""
echo "Testing requeue (playlist) mode..."
run_test "TestTaskRequeue" || ((failed++))

echo ""
echo "Testing lock expiration..."
run_test "TestCleanupExpiredLocks" || ((failed++))

echo ""
echo "=== Test Summary ==="
if [ $failed -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}$failed test(s) failed${NC}"
    echo ""
    echo "Run with verbose output:"
    echo "  go test -v ./..."
    exit 1
fi