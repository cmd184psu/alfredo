#!/bin/bash
# Integration test script for CTQ

set -e

COORDINATOR_PID=""
WORKER_PID=""
DB_PATH="/tmp/ctq-test-$$.sqlite"
HTTP_ADDR="127.0.0.1:18080"
CTQCTL="./ctqctl -url http://${HTTP_ADDR}"

cleanup() {
    echo ""
    echo "=== Cleaning up ==="
    if [ -n "$WORKER_PID" ]; then
        echo "Stopping worker (PID: $WORKER_PID)..."
        kill $WORKER_PID 2>/dev/null || true
    fi
    if [ -n "$COORDINATOR_PID" ]; then
        echo "Stopping coordinator (PID: $COORDINATOR_PID)..."
        kill $COORDINATOR_PID 2>/dev/null || true
    fi
    rm -f "$DB_PATH"
    echo "Cleanup complete"
}

trap cleanup EXIT

echo "=== Building CTQ ==="
make build

echo ""
echo "=== Starting Coordinator ==="
./ctq -mode coordinator -db "$DB_PATH" -http "$HTTP_ADDR" > /tmp/coordinator.log 2>&1 &
COORDINATOR_PID=$!
echo "Coordinator started (PID: $COORDINATOR_PID)"
sleep 2

echo ""
echo "=== Starting Worker ==="
./ctq -mode worker -db "$DB_PATH" -worker-id test-worker > /tmp/worker.log 2>&1 &
WORKER_PID=$!
echo "Worker started (PID: $WORKER_PID)"
sleep 2

echo ""
echo "=== Testing Health ==="
$CTQCTL health

echo ""
echo "=== Adding Test Tasks ==="

# Task 1: Simple echo task
echo '{"name":"echo-task","enabled":true,"priority":50,"cooldown_seconds":5,"max_retries":2,"requeue":true,"task_type":"shell","args":"{\"shell\":\"echo Task executed at $(date)\"}"}' | $CTQCTL add

# Task 2: Success task with file creation
echo '{"name":"create-file","enabled":true,"priority":40,"cooldown_seconds":10,"max_retries":1,"requeue":false,"task_type":"shell","args":"{\"shell\":\"echo success > /tmp/ctq-test-output.txt\"}"}' | $CTQCTL add

# Task 3: Failing task to test retries
echo '{"name":"failing-task","enabled":true,"priority":60,"cooldown_seconds":3,"max_retries":3,"requeue":false,"task_type":"shell","args":"{\"shell\":\"exit 1\"}"}' | $CTQCTL add

# Task 4: Low priority task
echo '{"name":"low-priority","enabled":true,"priority":200,"cooldown_seconds":5,"max_retries":0,"requeue":true,"task_type":"shell","args":"{\"shell\":\"echo Low priority task\"}"}' | $CTQCTL add

echo ""
echo "=== Listing Tasks ==="
$CTQCTL list

echo ""
echo "=== Waiting for tasks to execute (30 seconds) ==="
sleep 30

echo ""
echo "=== Checking Executions ==="
$CTQCTL executions

echo ""
echo "=== Checking Metrics ==="
$CTQCTL metrics

echo ""
echo "=== Testing Queue Pause ==="
$CTQCTL pause
$CTQCTL status
sleep 2

echo ""
echo "=== Testing Queue Resume ==="
$CTQCTL resume
$CTQCTL status

echo ""
echo "=== Testing Task Disable ==="
$CTQCTL disable -name failing-task
$CTQCTL list | grep failing-task

echo ""
echo "=== Checking Final State ==="
$CTQCTL executions -limit 20

echo ""
echo "=== Test Summary ==="
echo "Check /tmp/coordinator.log and /tmp/worker.log for detailed logs"
if [ -f /tmp/ctq-test-output.txt ]; then
    echo "✓ File creation task succeeded"
    cat /tmp/ctq-test-output.txt
    rm /tmp/ctq-test-output.txt
fi

echo ""
echo "=== All Tests Completed ==="