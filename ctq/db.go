package ctq

import (
	"fmt"
	"time"

	"github.com/cmd184psu/alfredo"
)

const schema = `
-- Task definitions
CREATE TABLE IF NOT EXISTS tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	enabled BOOLEAN NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 100,
	cooldown_seconds INTEGER NOT NULL DEFAULT 0,
	max_retries INTEGER NOT NULL DEFAULT 3,
	requeue BOOLEAN NOT NULL DEFAULT 0, -- playlist mode: return to queue
	task_type TEXT NOT NULL, -- e.g., 'exec', 'script', 'ssh'
	args TEXT NOT NULL, -- JSON encoded arguments
	created_at INTEGER NOT NULL DEFAULT 0,      -- epoch ms
	updated_at INTEGER NOT NULL DEFAULT 0
);

-- Task execution history and state
CREATE TABLE IF NOT EXISTS task_executions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id INTEGER NOT NULL,
	started_at INTEGER,
	finished_at INTEGER,
	status TEXT NOT NULL, -- 'pending', 'running', 'success', 'failed'
	error_message TEXT,
	retry_count INTEGER NOT NULL DEFAULT 0,
	worker_id TEXT,
	duration_ms INTEGER,
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Lock table for distributed locking
CREATE TABLE IF NOT EXISTS task_locks (
	task_id INTEGER PRIMARY KEY,
	worker_id TEXT NOT NULL,
	acquired_at INTEGER NOT NULL DEFAULT 0,
	expires_at INTEGER NOT NULL,                -- epoch ms
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Queue pause state
CREATE TABLE IF NOT EXISTS queue_state (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	paused BOOLEAN NOT NULL DEFAULT 0,
	paused_at INTEGER,
	paused_by TEXT
);

-- Metrics for observability
CREATE TABLE IF NOT EXISTS task_metrics (
	task_id INTEGER NOT NULL,
	recorded_at INTEGER NOT NULL,
	duration_ms INTEGER NOT NULL,
	status TEXT NOT NULL,
	PRIMARY KEY (task_id, recorded_at),
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tasks_enabled_priority ON tasks(enabled, priority, id);
CREATE INDEX IF NOT EXISTS idx_executions_task_status ON task_executions(task_id, status, finished_at);
CREATE INDEX IF NOT EXISTS idx_executions_finished ON task_executions(finished_at DESC);
CREATE INDEX IF NOT EXISTS idx_locks_expires ON task_locks(expires_at);
CREATE INDEX IF NOT EXISTS idx_metrics_task_time ON task_metrics(task_id, recorded_at DESC);

-- Initialize queue state
INSERT OR IGNORE INTO queue_state (id, paused) VALUES (1, 0);
`

type DB struct {
	*alfredo.DatabaseStruct
}

func InitDB(dbPath string) (*DB, error) {
	db := alfredo.NewSQLiteDB().WithDbPath(dbPath)

	if err := db.Query(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db}, nil
}

// const cleanupExpiredLocksFmt = `
// DELETE FROM task_locks
// WHERE expires_at < strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
// `

// func (db *DB) CleanupExpiredLocks() error {
// 	return db.Query(cleanupExpiredLocksFmt)
// }

// const cleanupExpiredLocksFmt = `
// UPDATE tasks
//         SET status = 'available'
//         FROM task_locks tl
//         WHERE tasks.id = tl.task_id
//           AND tl.expires_at < '%s'
//           AND tasks.status = 'locked'
//           AND tl.worker_id IS NOT NULL`

// func (db *DB) CleanupExpiredLocks() error {

// 	// RFC3339 now - Go-Lang compatible, no datetime()
// 	nowRFC3339 := time.Now().Format(time.RFC3339)
// 	return db.Query(fmt.Sprintf(cleanupExpiredLocksFmt, nowRFC3339))
// }

// const cleanupExpiredLocksFmtWTF = `
// DELETE FROM task_locks
// WHERE expires_at < strftime('%Y-%m-%d %H:%M:%S', 'now')
//   AND task_id IN (SELECT id FROM tasks WHERE status = 'locked')`

// this one works.. maybe?
const cleanupExpiredLocksFmt = `
DELETE FROM task_locks 
WHERE expires_at < {{now}}
  AND worker_id IS NOT NULL`

func (db *DB) CleanupExpiredLocks() error {
	return db.Query(cleanupExpiredLocksFmt)

	// if err := db.Query(cleanupExpiredLocksFmt); err != nil {
	// 	panic(fmt.Sprintf("CleanupExpiredLocks failed (1): %v", err))
	// 	return err
	// }
	// Also reset task status for cleaned locks
	//	resetFmt := "UPDATE tasks SET status = 'available' WHERE id NOT IN (SELECT task_id FROM task_locks WHERE worker_id IS NOT NULL) AND status = 'locked'"

	// resetFmt := `UPDATE tasks
	//             SET status = 'available'
	//             WHERE status = 'locked'
	//             AND NOT EXISTS (
	//                 SELECT 1 FROM task_locks
	//                 WHERE task_id = tasks.id AND worker_id IS NOT NULL
	//             )`

	// resetFmt := `UPDATE tasks
	//         SET status = 'available'
	//         WHERE status = 'locked'
	//         AND id NOT IN (
	//             SELECT task_id FROM task_locks
	//             WHERE worker_id IS NOT NULL
	//         )`

	// if err := db.Query(resetFmt); err != nil {
	// 	panic(fmt.Sprintf("Task reset failed: %v", err))
	// 	return err
	// }

	// // Return locks cleaned count

	// fmt.Println("Locks cleaned:", db.GetResultInt64())
	// return db.Query("SELECT COUNT(*) FROM task_locks WHERE expires_at  < {{now}}")
}

// func (db *DB) CleanupExpiredLocks() error {
//     const fmtStr = "DELETE FROM task_locks WHERE expires_at < {{now}} AND worker_id IS NOT NULL"
//     fmt.Println("DEBUG SQL:", fmtStr)  // Print exact SQL
//     return db.Query(fmtStr)
// }

// ORDER BY
//   t.priority ASC,           -- 1st: priority
//   CASE WHEN tl.task_id IS NOT NULL THEN 999 ELSE 0 END,  -- 2nd: locked tasks last
//   le.last_finished_at ASC,
//   (le.last_finished_at IS NULL)
// LIMIT 1;

const acquireLockFmt = `
INSERT INTO task_locks (task_id, worker_id, expires_at)
VALUES (%d, '%s', %d)
ON CONFLICT(task_id) DO NOTHING;
`

//SELECT changes();
//`

// func (db *DB) AcquireLock(taskID int64, workerID string, lockDuration time.Duration) (bool, error) {
// 	// Format duration as SQLite modifier (e.g. "5 minutes", "2 hours")
// 	//durationStr := lockDuration.Round(time.Second).String()
// 	durationStr := durationToSQLiteModifier(lockDuration)
// 	payload := fmt.Sprintf(acquireLockFmt,
// 		taskID,
// 		workerID,
// 		durationStr,
// 	)

// 	if err := db.Query(payload); err != nil {
// 		return false, err
// 	}

// 	return db.GetResultInt64() > 0, nil
// }

const checkActiveLockFmt = `
SELECT COUNT(*) FROM task_locks WHERE task_id = %d AND expires_at > {{now}}
`

func (db *DB) AcquireLock(taskID int64, workerID string, lockDuration time.Duration) (bool, error) {
	// 1. Check if active lock exists
	checkQuery := fmt.Sprintf(checkActiveLockFmt, taskID)
	if err := db.Query(checkQuery); err != nil {
		return false, err
	}
	if db.GetResultInt64() > 0 { // Lock exists
		return false, nil
	}

	// 2. INSERT new lock
	payload := fmt.Sprintf(acquireLockFmt, taskID, workerID, time.Now().Add(lockDuration).UnixMilli())
	if err := db.Query(payload); err != nil {
		return false, err
	}

	return true, nil // Always true if we reach here
}

const releaseLockFmt = `
DELETE FROM task_locks WHERE task_id = %d AND worker_id = '%s';
SELECT changes();
`

func (db *DB) ReleaseLock(taskID int64, workerID string) error {
	payload := fmt.Sprintf(releaseLockFmt, taskID, workerID)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	// Note: it's OK if 0 rows affected (lock already expired/removed)
	alfredo.VerbosePrintf("Released lock for task %d by worker %s (rows affected: %d)", taskID, workerID, rowsAffected)
	return nil
}

const isQueuePausedFmt = `
SELECT paused FROM queue_state WHERE id = 1;
`

func (db *DB) IsQueuePaused() bool {
	if err := db.Query(isQueuePausedFmt); err != nil {
		panic(fmt.Sprintf("IsQueuePaused query failed: %v", err))
	}

	result := db.GetResultInt64() // 1=true, 0=false
	return result == 1
}

const setQueuePausedFmt = `
UPDATE queue_state 
SET paused = %d, paused_at = {{now}}, paused_by = '%s'
WHERE id = 1;
SELECT changes();
`

func (db *DB) SetQueuePaused(paused bool, pausedBy string) error {
	var pausedInt int
	switch paused {
	case true:
		pausedInt = 1
	case false:
		pausedInt = 0
	}
	payload := fmt.Sprintf(setQueuePausedFmt,
		pausedInt,
		pausedBy,
	)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	if rowsAffected != 1 {
		return fmt.Errorf("queue state update affected %d rows (expected 1)", rowsAffected)
	}

	return nil
}
