package ctq

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestEndToEnd runs a complete workflow test
func TestEndToEnd(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")

	// Initialize database
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	t.Run("AddTask", func(t *testing.T) {
		task := &Task{
			Name:            "test-task",
			Enabled:         true,
			Priority:        50,
			CooldownSeconds: 5,
			MaxRetries:      3,
			Requeue:         true,
			TaskType:        "shell",
			Args:            `{"shell":"echo test"}`,
		}

		if err := db.AddTask(task); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}

		// Verify task was added
		retrieved, err := db.GetTask("test-task")
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Task not found after add")
		}
		if retrieved.Name != "test-task" {
			t.Errorf("Expected name 'test-task', got '%s'", retrieved.Name)
		}
		if !retrieved.Enabled {
			t.Error("Task should be enabled")
		}
	})

	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := db.ListTasks()
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
	})

	t.Run("GetNextTask_FirstRun", func(t *testing.T) {
		twe, err := db.GetNextTask()
		if err != nil {
			t.Fatalf("GetNextTask failed: %v", err)
		}
		if twe == nil {
			t.Fatal("Expected task, got nil")
		}
		if twe.Task.Name != "test-task" {
			t.Errorf("Expected 'test-task', got '%s'", twe.Task.Name)
		}
		if twe.LastExecution != nil {
			t.Error("First run should have no LastExecution")
		}
	})

	t.Run("AcquireLock", func(t *testing.T) {
		// Get task ID
		task, _ := db.GetTask("test-task")

		acquired, err := db.AcquireLock(task.ID, "worker-1", 5*time.Minute)
		if err != nil {
			t.Fatalf("AcquireLock failed: %v", err)
		}
		if !acquired {
			t.Error("Should have acquired lock")
		}

		// Try to acquire again (should fail)
		acquired2, err := db.AcquireLock(task.ID, "worker-2", 5*time.Minute)
		if err != nil {
			t.Fatalf("Second AcquireLock failed: %v", err)
		}
		if acquired2 {
			t.Error("Should not acquire lock twice")
		}
	})

	t.Run("CreateExecution", func(t *testing.T) {
		task, _ := db.GetTask("test-task")

		execID, err := db.CreateExecution(task.ID, "worker-1")
		if err != nil {
			t.Fatalf("CreateExecution failed: %v", err)
		}
		if execID == 0 {
			t.Error("Execution ID should not be 0")
		}
	})

	t.Run("UpdateExecution", func(t *testing.T) {
		task, _ := db.GetTask("test-task")
		execID, _ := db.CreateExecution(task.ID, "worker-1")

		errorMsg := "test error"
		err := db.UpdateExecution(execID, "failed", &errorMsg, 100)
		if err != nil {
			t.Fatalf("UpdateExecution failed: %v", err)
		}
	})

	t.Run("RecordMetric", func(t *testing.T) {
		db.AddTask(&Task{Name: "test-task", Enabled: true}) // Minimal task
		task, err := db.GetTask("test-task")
		require.NoError(t, err)
		require.NotNil(t, task)

		err = db.RecordMetric(task.ID, 123, "success")
		require.NoError(t, err)
	})

	t.Run("ReleaseLock", func(t *testing.T) {
		task, _ := db.GetTask("test-task")

		err := db.ReleaseLock(task.ID, "worker-1")
		if err != nil {
			t.Fatalf("ReleaseLock failed: %v", err)
		}

		// Should be able to acquire again
		acquired, err := db.AcquireLock(task.ID, "worker-2", 5*time.Minute)
		if err != nil {
			t.Fatalf("AcquireLock after release failed: %v", err)
		}
		if !acquired {
			t.Error("Should acquire lock after release")
		}
		db.ReleaseLock(task.ID, "worker-2")
	})

	t.Run("EnableDisableTask", func(t *testing.T) {
		// FIX: Create task FIRST
		db.AddTask(&Task{Name: "test-task", Enabled: true})

		// Disable
		if err := db.EnableTask("test-task", false); err != nil {
			t.Fatalf("DisableTask failed: %v", err)
		}

		task, err := db.GetTask("test-task")
		require.NoError(t, err)
		require.False(t, task.Enabled, "Task should be disabled")

		// Task should not be returned by GetNextTask when disabled
		twe, err := db.GetNextTask()
		require.NoError(t, err)
		require.Nil(t, twe, "Disabled task should not be returned")

		// Re-enable
		if err := db.EnableTask("test-task", true); err != nil {
			t.Fatalf("EnableTask failed: %v", err)
		}

		task, err = db.GetTask("test-task")
		require.NoError(t, err)
		require.True(t, task.Enabled, "Task should be enabled")
	})

	t.Run("QueuePauseResume", func(t *testing.T) {
		// Pause
		if err := db.SetQueuePaused(true, "test-user"); err != nil {
			t.Fatalf("SetQueuePaused(true) failed: %v", err)
		}

		if !db.IsQueuePaused() {
			t.Error("Queue should be paused")
		}

		// Resume
		if err := db.SetQueuePaused(false, ""); err != nil {
			t.Fatalf("SetQueuePaused(false) failed: %v", err)
		}

		if db.IsQueuePaused() {
			t.Error("Queue should not be paused")
		}
	})

	t.Run("RefreshTask", func(t *testing.T) {
		// Add a one-shot task
		oneShot := &Task{
			Name:            "one-shot",
			Enabled:         true,
			Priority:        10,
			CooldownSeconds: 0,
			MaxRetries:      0,
			Requeue:         false,
			TaskType:        "shell",
			Args:            `{"shell":"echo oneshot"}`,
		}
		db.AddTask(oneShot)

		// Execute it
		task, _ := db.GetTask("one-shot")
		execID, _ := db.CreateExecution(task.ID, "worker-1")
		db.UpdateExecution(execID, "success", nil, 50)
		db.RecordMetric(task.ID, 50, "success")

		// Should NOT be returned by GetNextTask (requeue=false and already ran)
		twe, _ := db.GetNextTask()
		if twe != nil && twe.Task.Name == "one-shot" {
			t.Error("One-shot task should not run again before refresh")
		}

		// Refresh it
		if err := db.RefreshTask("one-shot"); err != nil {
			t.Fatalf("RefreshTask failed: %v", err)
		}

		// Now it should be available again
		twe, err := db.GetNextTask()
		if err != nil {
			t.Fatalf("GetNextTask after refresh failed: %v", err)
		}
		if twe == nil {
			t.Fatal("Expected task after refresh")
		}
		if twe.Task.Name != "one-shot" {
			t.Errorf("Expected 'one-shot', got '%s'", twe.Task.Name)
		}
		if twe.LastExecution != nil {
			t.Error("After refresh, LastExecution should be nil")
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		if err := db.DeleteTask("test-task"); err != nil {
			t.Fatalf("DeleteTask failed: %v", err)
		}

		task, err := db.GetTask("test-task")
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}
		if task != nil {
			t.Error("Task should be deleted")
		}
	})
}

// TestTaskPriority tests that tasks are returned in priority order
func TestTaskPriority(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Add tasks with different priorities
	tasks := []*Task{
		{Name: "low", Enabled: true, Priority: 100, CooldownSeconds: 0, MaxRetries: 0, Requeue: true, TaskType: "shell", Args: `{"shell":"echo low"}`},
		{Name: "high", Enabled: true, Priority: 10, CooldownSeconds: 0, MaxRetries: 0, Requeue: true, TaskType: "shell", Args: `{"shell":"echo high"}`},
		{Name: "medium", Enabled: true, Priority: 50, CooldownSeconds: 0, MaxRetries: 0, Requeue: true, TaskType: "shell", Args: `{"shell":"echo medium"}`},
	}

	for _, task := range tasks {
		if err := db.AddTask(task); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	// GetNextTask should return highest priority (lowest number)
	twe, err := db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe == nil {
		t.Fatal("Expected task")
	}
	if twe.Task.Name != "high" {
		t.Errorf("Expected 'high' priority task first, got '%s'", twe.Task.Name)
	}
}

// TestTaskCooldown tests that cooldown prevents re-execution
func TestTaskCooldown(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Add task with 10 second cooldown
	task := &Task{
		Name:            "cooldown-test",
		Enabled:         true,
		Priority:        50,
		CooldownSeconds: 10,
		MaxRetries:      0,
		Requeue:         true,
		TaskType:        "shell",
		Args:            `{"shell":"echo test"}`,
	}
	db.AddTask(task)

	// First execution
	retrieved, _ := db.GetTask("cooldown-test")
	execID, _ := db.CreateExecution(retrieved.ID, "worker-1")
	db.UpdateExecution(execID, "success", nil, 100)

	// Should NOT be available immediately (cooldown not passed)
	twe, err := db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe != nil && twe.Task.Name == "cooldown-test" {
		t.Error("Task should not be available during cooldown")
	}
}

// TestTaskRetry tests retry logic
func TestTaskRetry(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Add task with retries but no requeue
	task := &Task{
		Name:            "retry-test",
		Enabled:         true,
		Priority:        50,
		CooldownSeconds: 0,
		MaxRetries:      3,
		Requeue:         false,
		TaskType:        "shell",
		Args:            `{"shell":"exit 1"}`,
	}
	db.AddTask(task)

	retrieved, _ := db.GetTask("retry-test")

	// First failure
	execID1, err := db.CreateExecution(retrieved.ID, "worker-1")
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}
	errMsg1 := "failed"
	db.UpdateExecution(execID1, "failed", &errMsg1, 50)

	// Should be available for retry
	twe, err := db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe == nil {
		t.Fatal("Task should be available for retry")
	}
	if twe.Task.Name != "retry-test" {
		t.Errorf("Expected 'retry-test', got '%s'", twe.Task.Name)
	}

	// Second failure
	execID2, err := db.CreateExecution(retrieved.ID, "worker-1")
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}
	db.UpdateExecution(execID2, "failed", &errMsg1, 50)

	// Still available (retry_count=1, max=3)
	twe, err = db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe == nil {
		t.Error("Task should still be available for retry")
	}

	// Third failure
	execID3, err := db.CreateExecution(retrieved.ID, "worker-1")
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}
	err = db.UpdateExecution(execID3, "failed", &errMsg1, 50)
	if err != nil {
		t.Fatalf("UpdateExecution failed: %v", err)
	}

	// Still available (retry_count=2, max=3)
	twe, err = db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe == nil {
		t.Error("Task should still be available for final retry")
	}

	// Fourth failure - exhausted retries
	execID4, err := db.CreateExecution(retrieved.ID, "worker-1")
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}
	err = db.UpdateExecution(execID4, "failed", &errMsg1, 50)
	if err != nil {
		t.Fatalf("UpdateExecution failed: %v", err)
	}

	// Should NOT be available (retries exhausted, no requeue)
	twe, err = db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe != nil && twe.Task.Name == "retry-test" {
		t.Error("Task should not be available after exhausting retries")
	}
}

// TestTaskRequeue tests playlist mode
func TestTaskRequeue(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Add task with requeue=true and no cooldown
	task := &Task{
		Name:            "requeue-test",
		Enabled:         true,
		Priority:        50,
		CooldownSeconds: 0,
		MaxRetries:      0,
		Requeue:         true,
		TaskType:        "shell",
		Args:            `{"shell":"echo test"}`,
	}
	db.AddTask(task)

	retrieved, _ := db.GetTask("requeue-test")

	// Execute successfully
	execID, _ := db.CreateExecution(retrieved.ID, "worker-1")
	db.UpdateExecution(execID, "success", nil, 100)

	// Should be available again immediately (requeue=true, cooldown=0)
	twe, err := db.GetNextTask()
	if err != nil {
		t.Fatalf("GetNextTask failed: %v", err)
	}
	if twe == nil {
		t.Fatal("Task should be available again (requeue mode)")
	}
	if twe.Task.Name != "requeue-test" {
		t.Errorf("Expected 'requeue-test', got '%s'", twe.Task.Name)
	}
}

func checkLockCount(db *DB, taskID int64) (int, error) {
	err := db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id = %d", taskID))
	return db.GetResultInt(), err
}

// TestCleanupExpiredLocks tests lock expiration
// func TestCleanupExpiredLocks(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	dbPath := filepath.Join(tmpDir, "test.sqlite")
// 	db, err := InitDB(dbPath)
// 	if err != nil {
// 		t.Fatalf("Failed to init DB: %v", err)
// 	}

// 	task := &Task{
// 		Name:            "lock-test",
// 		Enabled:         true,
// 		Priority:        50,
// 		CooldownSeconds: 0,
// 		MaxRetries:      0,
// 		Requeue:         true,
// 		TaskType:        "shell",
// 		Args:            `{"shell":"echo test"}`,
// 	}
// 	db.AddTask(task)
// 	retrieved, err := db.GetTask("lock-test")
// 	if err != nil {
// 		t.Fatalf("GetTask failed: %v", err)
// 	}
// 	dur := 3 * time.Second
// 	// Acquire lock with very short duration
// 	acquired, err2 := db.AcquireLock(retrieved.ID, "worker-1", dur)
// 	if !acquired {

// 		t.Logf("AcquireLock DEBUG: task=%v acquired=%v", retrieved, acquired)
// 		t.Logf("AcquireLock FAIL reason: SQL='%s' response='%s'",
// 			fmt.Sprintf(acquireLockFmt, retrieved.ID, "worker-1", dur.Milliseconds()),
// 			db.GetResult())

// 		t.Fatal("Should have acquired lock")
// 	}
// 	if err2 != nil {
// 		t.Fatalf("AcquireLock failed: %v", err2)
// 	}

// 	db.Query(fmt.Sprintf("SELECT task_id, worker_id, expires_at FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	t.Logf("FULL LOCK ROW: %s", db.GetResult())

// 	// Wait for expiration
// 	time.Sleep(dur + 2*time.Second)

// 	t.Logf("=== Before Cleanup ===")
// 	count1, _ := checkLockCount(db, retrieved.ID)
// 	t.Logf("Locks before: %d", count1)

// 	if err := db.Query(fmt.Sprintf("SELECT expires_at FROM task_locks WHERE task_id=%d", retrieved.ID)); err != nil {
// 		t.Fatalf("Query expires_at failed: %v", err)
// 	}
// 	t.Logf("EXPIRES_AT: %s", db.GetResult())

// 	if err := db.CleanupExpiredLocks(); err != nil {
// 		t.Fatalf("CleanupExpiredLocks failed: %v", err)
// 	}
// 	t.Logf("CLEANUP NOW: %s", db.GetResult())

// 	// Check what cleanup would delete WITHOUT deleting

// 	selectCountFmt := `SELECT COUNT(*) FROM task_locks
// 					  WHERE expires_at < {{now}}
// 					  AND worker_id IS NOT NULL`
// 	if err := db.Query(selectCountFmt); err != nil {
// 		t.Fatalf("Select count failed: %v", err)
// 	}
// 	t.Logf("WOULD DELETE: %s", db.GetResult())

// 	// Cleanup
// 	if err := db.CleanupExpiredLocks(); err != nil {
// 		t.Fatalf("CleanupExpiredLocks failed: %v", err)
// 	}
// 	// I don't think that's how this query works
// 	rowsAffected := db.GetResultInt64()
// 	t.Logf("Cleanup affected: %d rows", rowsAffected)
// 	require.Equal(t, int64(1), rowsAffected)

// 	// // Verify reclaim
// 	//task table does not have status, can't do it
// 	// if err := db.Query(fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE id=%d AND status='available'", retrieved.ID)); err != nil {
// 	// 	t.Fatalf("Query available tasks failed: %v", err)
// 	// }
// 	// availableCount := db.GetResultInt64()
// 	// t.Logf("Locks for task %d after cleanup: %d", retrieved.ID, availableCount)
// 	// require.Equal(t, int64(1), availableCount)

// 	// After db.CleanupExpiredLocks()
// 	countQuery := "SELECT COUNT(*) FROM task_locks WHERE task_id = %d"
// 	if err := db.Query(fmt.Sprintf(countQuery, retrieved.ID)); err != nil {
// 		t.Fatalf("Count query failed: %v", err)
// 	}

// 	t.Logf("== raw result from db.GetResult(): %s", db.GetResult())
// 	t.Logf("== result formmatted as an int64: %d", db.GetResultInt64())
// 	t.Logf("== result formmatted as an int: %d", db.GetResultInt())

// 	lockCount, _ := strconv.Atoi(db.GetResult())
// 	t.Logf("Locks for task %d after cleanup: %d", retrieved.ID, lockCount)

// 	// Should be able to acquire again
// 	acquired2, err := db.AcquireLock(retrieved.ID, "worker-2", 5*time.Minute)
// 	if err != nil {
// 		t.Fatalf("Second acquire failed: %v", err)
// 	}
// 	if !acquired2 {
// 		t.Error("Should acquire lock after cleanup")
// 	}
// }

// Add to your DB struct (temporary for testing)
// func (db *DB) CleanupExpiredLocksWithCount() (int, error) {
// 	err := db.Query(cleanupExpiredLocksFmt)
// 	if err != nil {
// 		return 0, err
// 	}
// 	result := db.GetResult()
// 	// Parse affected rows from result (adjust based on your output format)
// 	affected, _ := strconv.Atoi(strings.Trim(result, " \n"))
// 	return affected, nil
// }

// func countLocks(t *testing.T, db *DB, taskID int64) int {
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", taskID))
// 	count, _ := strconv.Atoi(db.GetResult())
// 	return count
// }

// func getLockExpiresAt(t *testing.T, db *DB, taskID int64) int64 {
// 	db.Query(fmt.Sprintf("SELECT expires_at FROM task_locks WHERE task_id=%d", taskID))
// 	expiresAt, _ := strconv.ParseInt(db.GetResult(), 10, 64)
// 	return expiresAt
// }

func TestCleanupExpiredLocks(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	db, err := InitDB(dbPath)
	require.NoError(t, err)

	// Create task
	task := &Task{
		Name:            "lock-test",
		Enabled:         true,
		Priority:        50,
		CooldownSeconds: 0,
		MaxRetries:      0,
		Requeue:         true,
		TaskType:        "shell",
		Args:            `{"shell":"echo test"}`,
	}
	db.AddTask(task)
	retrieved, err := db.GetTask("lock-test")
	require.NoError(t, err)

	// 1. Acquire lock (3s expiry)
	dur := 3 * time.Millisecond
	acquired, err := db.AcquireLock(retrieved.ID, "worker-1", dur)
	require.NoError(t, err)
	require.True(t, acquired)

	// 2. Verify lock + expiry
	db.Query(fmt.Sprintf("SELECT task_id,worker_id,expires_at FROM task_locks WHERE task_id=%d", retrieved.ID))
	t.Logf("LOCK: %s", db.GetResult())

	db.Query(fmt.Sprintf("SELECT expires_at FROM task_locks WHERE task_id=%d", retrieved.ID))
	expiresAt, _ := strconv.ParseInt(db.GetResult(), 10, 64)
	t.Logf("EXPIRES_AT: %d", expiresAt)

	// 3. Wait past expiry
	time.Sleep(dur + 2*time.Millisecond)
	now := time.Now().UnixMilli()
	t.Logf("CLEANUP_NOW: %d (expired by %dms)", now, now-expiresAt)

	// 4. Verify expired BEFORE cleanup
	db.Query(`SELECT COUNT(*) FROM task_locks WHERE expires_at < {{now}} AND worker_id IS NOT NULL`)
	wouldDelete, _ := strconv.Atoi(db.GetResult())
	require.Equal(t, 1, wouldDelete)
	t.Logf("WOULD_DELETE: %d", wouldDelete)

	// 5. Count BEFORE cleanup
	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
	beforeCount, _ := strconv.Atoi(db.GetResult())
	require.Equal(t, 1, beforeCount)
	t.Logf("BEFORE_CLEANUP: %d", beforeCount)

	// 6. Cleanup
	err = db.CleanupExpiredLocks()
	require.NoError(t, err)

	// 7. Count AFTER cleanup
	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
	afterCount, _ := strconv.Atoi(db.GetResult())
	require.Equal(t, 0, afterCount)
	t.Logf("AFTER_CLEANUP: %d", afterCount)

	// 8. Verify can re-acquire
	acquired2, err := db.AcquireLock(retrieved.ID, "worker-2", 5*time.Minute)
	require.NoError(t, err)
	require.True(t, acquired2)
}

// func TestStrftimeLockCleanupFails(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	dbPath := filepath.Join(tmpDir, "test.sqlite")
// 	db, err := InitDB(dbPath)
// 	require.NoError(t, err)

// 	// Create task using EXACT same pattern as working test
// 	task := &Task{
// 		Name:            "strftime-test",
// 		Enabled:         true,
// 		Priority:        50,
// 		CooldownSeconds: 0,
// 		MaxRetries:      0,
// 		Requeue:         true,
// 		TaskType:        "shell",
// 		Args:            `{"shell":"echo test"}`,
// 	}
// 	db.AddTask(task)
// 	retrieved, err := db.GetTask("strftime-test")
// 	require.NoError(t, err)

// 	// Create expired lock (10s ago epoch ms)
// 	expiredAt := time.Now().UnixMilli() - 10000 // 10s expired
// 	db.Query(fmt.Sprintf(`INSERT INTO task_locks (task_id, worker_id, expires_at)
// 	                     VALUES (%d, 'worker-1', %d)`, retrieved.ID, expiredAt))

// 	// Verify lock exists
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	count, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 1, count)

// 	// TEST 1: strftime TEXT vs INTEGER should FAIL
// 	err = db.Query("DELETE FROM task_locks WHERE expires_at < strftime('%s','now')*1000 AND worker_id IS NOT NULL")
// 	// Don't fatal here - we expect this to fail silently (0 rows)

// 	// Verify lock STILL EXISTS (proof strftime failed)
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	failedCount, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 1, failedCount, "strftime cleanup FAILED - lock still exists (CORRECT)")
// 	t.Logf("STRFTIME_CLEANUP: %d locks remain (should fail silently)", failedCount)

// 	// TEST 2: Your working {{now}} path succeeds
// 	err = db.CleanupExpiredLocks()
// 	require.NoError(t, err)

// 	// Verify lock GONE
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	successCount, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 0, successCount, "epoch {{now}} cleanup SUCCEEDED")
// 	t.Logf("EPOCH_CLEANUP: %d locks remain (should be 0)", successCount)
// }

// func TestProductionStrftimeBugs(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	dbPath := filepath.Join(tmpDir, "test.sqlite")
// 	db, err := InitDB(dbPath)
// 	require.NoError(t, err)

// 	// Create task (EXACT copy from your working test)
// 	task := &Task{
// 		Name:            "strftime-bug",
// 		Enabled:         true,
// 		Priority:        50,
// 		CooldownSeconds: 0,
// 		MaxRetries:      0,
// 		Requeue:         true,
// 		TaskType:        "shell",
// 		Args:            `{"shell":"echo test"}`,
// 	}
// 	db.AddTask(task)
// 	retrieved, err := db.GetTask("strftime-bug")
// 	require.NoError(t, err)

// 	// Create expired lock (10s ago)
// 	expiredAt := time.Now().UnixMilli() - 10000
// 	db.Query(fmt.Sprintf(`INSERT INTO task_locks (task_id, worker_id, expires_at)
// 	                     VALUES (%d, 'worker-1', %d)`, retrieved.ID, expiredAt))

// 	// Verify lock exists
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	count, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 1, count)

// 	// TEST YOUR ACTUAL PRODUCTION BUG: TEXT vs INTEGER comparison
// 	err = db.Query(fmt.Sprintf(`DELETE FROM task_locks WHERE
// 	                       expires_at < strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', 'now')
// 	                       AND task_id=%d`, retrieved.ID))

// 	// Lock should STILL EXIST (1770787752546 < "2026-02-11T15:20:00Z" = FALSE)
// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	failedCount, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 1, failedCount, "PRODUCTION strftime TEXT-vs-INTEGER FAILS (lock remains)")
// 	t.Logf("STRFTIME_BUG: %d locks remain - TEXT vs INTEGER comparison failed ✓", failedCount)

// 	// PROOF: Your {{now}} path works
// 	err = db.CleanupExpiredLocks()
// 	require.NoError(t, err)

// 	db.Query(fmt.Sprintf("SELECT COUNT(*) FROM task_locks WHERE task_id=%d", retrieved.ID))
// 	successCount, _ := strconv.Atoi(db.GetResult())
// 	require.Equal(t, 0, successCount, "{{now}} epoch cleanup WORKS")
// }

// Run with: go test -v -run TestEndToEnd
// Or: go test -v ./...
