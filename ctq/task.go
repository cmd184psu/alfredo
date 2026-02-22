package ctq

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cmd184psu/alfredo"
)

// Task represents a task definition
type Task struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	Enabled         bool              `json:"enabled"`
	Priority        int               `json:"priority"`
	CooldownSeconds int               `json:"cooldown_seconds"`
	MaxRetries      int               `json:"max_retries"`
	Requeue         bool              `json:"requeue"`
	TaskType        string            `json:"task_type"`
	Args            string            `json:"args"` // JSON string
	CreatedAt       alfredo.EpochTime `json:"created_at"`
	UpdatedAt       alfredo.EpochTime `json:"updated_at"`
}

func (t *Task) IsEnabled() bool {
	return t.Enabled
}

func (t *Task) ShouldRequeue() bool {
	return t.Requeue
}

// func (t *Task) LoadFromSlice(fields []string) error {
// 	if len(fields) < 11 {
// 		return fmt.Errorf("not enough fields to load Task")
// 	}
// 	t.ID = alfredo.Atoi64(fields[0], 0)
// 	t.Name = fields[1]
// 	t.Enabled = fields[2] == "1"
// 	t.Priority = alfredo.Atoi(fields[3], 0)
// 	t.CooldownSeconds = alfredo.Atoi(fields[4], 0)
// 	t.MaxRetries = alfredo.Atoi(fields[5], 0)
// 	t.Requeue = fields[6] == "1"
// 	t.TaskType = fields[7]
// 	t.Args = fields[8]
// 	createdAtStr := fields[9]
// 	updatedAtStr := fields[10]
// 	layout := "2006-01-02 15:04:05"
// 	createdAt, err := time.Parse(layout, createdAtStr)
// 	if err != nil {
// 		return fmt.Errorf("parse created_at: %w", err)
// 	}
// 	updatedAt, err := time.Parse(layout, updatedAtStr)
// 	if err != nil {
// 		return fmt.Errorf("parse updated_at: %w", err)
// 	}
// 	t.CreatedAt = createdAt
// 	t.UpdatedAt = updatedAt
// 	return nil
// }

// TaskExecution represents a single execution of a task
type TaskExecution struct {
	ID           int64      `json:"id"`
	TaskID       int64      `json:"task_id"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message"`
	RetryCount   int        `json:"retry_count"`
	WorkerID     *string    `json:"worker_id"`
	DurationMs   *int64     `json:"duration_ms"`
}

// TaskWithExecution combines task and execution info
type TaskWithExecution struct {
	Task          Task
	LastExecution *TaskExecution
}

// const getNextTaskFmt = `
// WITH latest_executions AS (
//     SELECT
//         te.task_id,
//         MAX(te.finished_at) as last_finished_at,
//         -- Simple: status from the latest row
//         last_status.status as status,
//         -- Simple: total failed count
//         total_fails.retry_count
//     FROM task_executions te
//     CROSS JOIN (SELECT task_id, COUNT(*) as retry_count
//                 FROM task_executions
//                 WHERE status = 'failed'
//                 GROUP BY task_id) total_fails
//         ON total_fails.task_id = te.task_id
//     CROSS JOIN (SELECT task_id, status
//                 FROM task_executions te2
//                 WHERE finished_at = (SELECT MAX(finished_at)
//                                    FROM task_executions te3
//                                    WHERE te3.task_id = te2.task_id)) last_status
//         ON last_status.task_id = te.task_id
//     WHERE te.status IN ('success', 'failed')
//     GROUP BY te.task_id
// )
// SELECT
//     json_object(
//         'id', t.id,
//         'name', t.name,
//         'enabled', CASE WHEN t.enabled = 1 THEN json('true') ELSE json('false') END,                    -- 1→JSON true, 0→JSON false ✓
//         'priority', t.priority,
//         'cooldown_seconds', t.cooldown_seconds,
//         'max_retries', t.max_retries,
//         'requeue', CASE WHEN t.requeue = 1 THEN json('true') ELSE json('false') END,                    -- 1→JSON true, 0→JSON false ✓
//         'task_type', t.task_type,
//         'args', t.args,
// 		'created_at', t.created_at,
// 		'updated_at', t.updated_at,
//         'last_finished_at', le.last_finished_at, -- NULL→JSON null ✓
//         'status', le.status,
//         'retry_count', le.retry_count
//     )
// FROM tasks t
// LEFT JOIN latest_executions le ON t.id = le.task_id
// LEFT JOIN task_locks tl ON t.id = tl.task_id
// WHERE t.enabled = 1
//   AND tl.task_id IS NULL
//   AND (
//       le.last_finished_at IS NULL
//       OR (
// 		1709289600 + (strftime('%s', 'now') - 1709289600) -
// 		CAST(strftime('%s', le.last_finished_at) AS INTEGER) >= t.cooldown_seconds
//           AND (
//               t.requeue = 1
//               OR (le.status = 'failed' AND le.retry_count <= t.max_retries)
//           )
//       )
//   )
// ORDER BY
//   t.priority ASC,
//   le.last_finished_at ASC,
//   (le.last_finished_at IS NULL)
// LIMIT 1;
// `

const getNextTaskFmt = `
WITH latest_executions AS (
    SELECT 
        task_id,
        MAX(finished_at) as last_finished_at,
        SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as retry_count,
        MAX(CASE WHEN finished_at = (SELECT MAX(finished_at) FROM task_executions te2 WHERE te2.task_id = task_executions.task_id) 
                 THEN status END) as status
    FROM task_executions
    WHERE status IN ('success', 'failed')
    GROUP BY task_id
)
SELECT 
    json_object(
        'id', t.id,
        'name', t.name,
        'enabled', CASE WHEN t.enabled = 1 THEN json('true') ELSE json('false') END,
        'priority', t.priority,
        'cooldown_seconds', t.cooldown_seconds,
        'max_retries', t.max_retries,
        'requeue', CASE WHEN t.requeue = 1 THEN json('true') ELSE json('false') END,
        'task_type', t.task_type,
        'args', t.args,
        'created_at', t.created_at,
        'updated_at', t.updated_at,
        'last_finished_at', le.last_finished_at,
        'status', le.status,
        'retry_count', le.retry_count
    )
FROM tasks t
LEFT JOIN latest_executions le ON t.id = le.task_id
LEFT JOIN task_locks tl ON t.id = tl.task_id
WHERE t.enabled = 1
  AND tl.task_id IS NULL
  AND (
      le.last_finished_at IS NULL
      OR (
          {{now}} - le.last_finished_at >= t.cooldown_seconds * 1000
          AND (
              t.requeue = 1
              OR (le.status = 'failed' AND le.retry_count <= t.max_retries)
          )
      )
  )
ORDER BY 
  t.priority ASC, 
  CASE WHEN tl.task_id IS NOT NULL THEN 999 ELSE 0 END,
  le.last_finished_at ASC, 
  (le.last_finished_at IS NULL)
LIMIT 1;
`

// ORDER BY
//   t.priority ASC,
//   le.last_finished_at ASC,
//   (le.last_finished_at IS NULL)
// LIMIT 1;
// `

// const GetNextTaskFmtbroken = `
// WITH latest_executions AS (
//     SELECT
//         task_id,
//         MAX(finished_at) as last_finished_at,
//         status,
//         retry_count
//     FROM task_executions
//     WHERE status IN ('success', 'failed')
//     GROUP BY task_id
// )
// SELECT
//     json_object(
//         'id', t.id,
//         'name', t.name,
//         'enabled', CASE WHEN t.enabled = 1 THEN json('true') ELSE json('false') END,                    -- 1→JSON true, 0→JSON false ✓
//         'priority', t.priority,
//         'cooldown_seconds', t.cooldown_seconds,
//         'max_retries', t.max_retries,
//         'requeue', CASE WHEN t.requeue = 1 THEN json('true') ELSE json('false') END,                    -- 1→JSON true, 0→JSON false ✓
//         'task_type', t.task_type,
//         'args', t.args,
// 		'created_at', t.created_at,
// 		'updated_at', t.updated_at,
//         'last_finished_at', le.last_finished_at, -- NULL→JSON null ✓
//         'status', le.status,
//         'retry_count', le.retry_count
//     )
// FROM tasks t
// LEFT JOIN latest_executions le ON t.id = le.task_id
// LEFT JOIN task_locks tl ON t.id = tl.task_id
// WHERE t.enabled = 1
//   AND tl.task_id IS NULL
//   AND (
//       le.last_finished_at IS NULL
//       OR (
// 		1709289600 + (strftime('%s', 'now') - 1709289600) -
// 		CAST(strftime('%s', le.last_finished_at) AS INTEGER) >= t.cooldown_seconds
//           AND (
//               t.requeue = 1
//               OR (le.status = 'failed' AND le.retry_count < t.max_retries)
//           )
//       )
//   )
// ORDER BY
//   t.priority ASC,
//   le.last_finished_at ASC,
//   (le.last_finished_at IS NULL)
// LIMIT 1;
// `

func (db *DB) GetNextTask() (*TaskWithExecution, error) {
	fmt.Println("BEGIN GetNextTask()")
	defer fmt.Println("END GetNextTask()")
	alfredo.SetVerbose(true)
	if err := db.Query(getNextTaskFmt); err != nil {
		fmt.Println("[db] GetNextTask: query error:", err)
		panic("query error: " + err.Error())
		return nil, err
	}

	alfredo.VerbosePrintln("===================")
	alfredo.VerbosePrintln(getNextTaskFmt)
	alfredo.VerbosePrintln("===================")
	alfredo.VerbosePrintln(db.GetResult())
	alfredo.VerbosePrintln("===================")
	if strings.Contains(db.GetResult(), "|") {
		panic("malformed result")
	}

	// Since this is nullable fields + struct composition, you have a few options:

	// Option 1: Wrapper already converts to JSON matching TaskWithExecution
	var twe TaskWithExecution
	if len(db.GetResult()) == 0 {
		fmt.Println("[db] GetNextTask: no tasks available (empty result)")
		return nil, nil // no rows
	}

	// if err := twe.Task.LoadFromSlice(db.GetResultAsSlice()); err != nil {
	// 	panic("failed to load Task from slice: " + err.Error())
	// 	return nil, err
	// }

	if err := json.Unmarshal([]byte(db.GetResult()), &twe.Task); err != nil {
		panic("failed to unmarshal Task JSON: " + err.Error())
		return nil, fmt.Errorf("failed to unmarshal TaskWithExecution JSON: %w", err)
	}
	return &twe, nil
}

// buildTaskExecutionPayload should:
// 1) Do the INSERT (either SELECT-based or simple VALUES).
// 2) End with: SELECT last_insert_rowid();
// func buildTaskExecutionPayload(taskID int64, workerID string, started time.Time) string {
// 	startedStr := started.UTC().Format(time.RFC3339Nano)

// 	return fmt.Sprintf(`
// INSERT INTO task_executions (task_id, started_at, status, worker_id, retry_count)
// VALUES (%d, '%s', '%s', 0);
// SELECT last_insert_rowid();
// `, taskID, startedStr, workerID)
// }

const InsertTaskExecutionFmt = `
INSERT INTO task_executions (task_id, started_at, status, worker_id, retry_count)
VALUES (%d, {{now}}, 'running', '%s', 0);
SELECT last_insert_rowid();
`

func (db *DB) CreateExecution(taskID int64, workerID string) (int64, error) {
	payload := fmt.Sprintf(InsertTaskExecutionFmt,
		taskID,
		workerID,
	)

	if err := db.Query(payload); err != nil {
		return 0, err
	}

	lastID := db.GetResultInt64()
	if lastID <= 0 {
		return 0, fmt.Errorf("insert failed, last_insert_rowid() = %d", lastID)
	}

	return lastID, nil
}

const UpdateTaskStatusFmt = `
UPDATE task_executions
SET finished_at = {{now}},
    status = '%s',
    error_message = %s,
    duration_ms = %d
WHERE id = %d;
SELECT changes();
`

func (db *DB) UpdateExecution(executionID int64, status string, errorMsg *string, durationMs int64) error {
	var errorMsgStr string
	if errorMsg != nil {
		errorMsgStr = fmt.Sprintf("'%s'", *errorMsg) // Safe quoting
	} else {
		errorMsgStr = "NULL" // SQLite NULL
	}

	payload := fmt.Sprintf(UpdateTaskStatusFmt,
		status,
		errorMsgStr,
		durationMs,
		executionID,
	)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	if rowsAffected != 1 {
		return fmt.Errorf("update affected %d rows (expected 1)", rowsAffected)
	}

	return nil
}

const recordMetricFmt = `
INSERT INTO task_metrics (task_id, duration_ms, status, recorded_at)
VALUES (%d, %d, '%s', {{now}});
SELECT changes();
`

func (db *DB) RecordMetric(taskID int64, durationMs int64, status string) error {
	fmt.Println("BEGIN RecordMetric()")
	defer fmt.Println("END RecordMetric()")
	payload := fmt.Sprintf(recordMetricFmt,
		taskID,
		durationMs,
		status)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	if rowsAffected != 1 {
		return fmt.Errorf("metric insert affected %d rows (expected 1)", rowsAffected)
	}

	return nil
}

const addTaskFmt = `
INSERT INTO tasks (name, enabled, priority, cooldown_seconds, max_retries, requeue, task_type, args, created_at, updated_at)
VALUES ('%s', %d, %d, %d, %d, %d, '%s', '%s', %d, %d)
ON CONFLICT(name) DO UPDATE SET
    enabled = excluded.enabled,
    priority = excluded.priority,
    cooldown_seconds = excluded.cooldown_seconds,
    max_retries = excluded.max_retries,
    requeue = excluded.requeue,
    task_type = excluded.task_type,
    args = excluded.args,
    updated_at = excluded.updated_at;
SELECT changes();
`

// Helper to convert bool to int for SQLite (0 or 1)
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (db *DB) AddTask(task *Task) error {

	// Set timestamps as EpochTime (your wrapper)
	if task.CreatedAt.IsZero() { // Check if zero time
		task.CreatedAt.Now()
	}
	task.UpdatedAt=task.CreatedAt

	// Execute - your Query handles SELECT changes() result
	return db.Query(fmt.Sprintf(addTaskFmt,
		task.Name, btoi(task.Enabled), task.Priority, task.CooldownSeconds,
		task.MaxRetries, btoi(task.Requeue), task.TaskType, task.Args,
		task.CreatedAt.UnixMilli(), task.UpdatedAt.UnixMilli()))
}

// func (db *DB) AddTask(task *Task) error {
//     now := time.Now().UnixMilli()

//     // Set timestamps before insert
//     if task.CreatedAt == 0 {
//         task.CreatedAt = now
//     }
//     task.UpdatedAt = now

//     changes := db.Query(fmt.Sprintf(addTaskFmt,
//         task.Name, boolInt(task.Enabled), task.Priority, task.CooldownSeconds,
//         task.MaxRetries, boolInt(task.Requeue), task.TaskType, task.Args,
//         task.CreatedAt, task.UpdatedAt))

//     if changes != nil {
//         result := db.GetResultInt64()

// func (db *DB) AddTask(task *Task) error {
// 	fmt.Printf("BEGIN AddTask(%s)\n", task.Name)
// 	defer fmt.Printf("END AddTask(%s)\n", task.Name)
// 	// Validate args is valid JSON
// 	var js map[string]interface{}
// 	if err := json.Unmarshal([]byte(task.Args), &js); err != nil {
// 		return fmt.Errorf("invalid JSON in args: %w", err)
// 	}

// 	payload := fmt.Sprintf(addTaskFmt,
// 		task.Name,
// 		btoi(task.Enabled),
// 		task.Priority,
// 		task.CooldownSeconds,
// 		task.MaxRetries,
// 		btoi(task.Requeue),
// 		task.TaskType,
// 		strings.ReplaceAll(task.Args, "'", "''"),
// 	)

// 	if err := db.Query(payload); err != nil {
// 		return err
// 	}

// 	rowsAffected := db.GetResultInt64()
// 	if rowsAffected == 0 {
// 		return fmt.Errorf("task operation affected 0 rows")
// 	}

// 	return nil
// }

const getTaskFmt = `
SELECT json_object(
  'id', id,
  'name', name,
  'enabled', CASE WHEN enabled = 1 THEN json('true') ELSE json('false') END,
  'priority', priority,
  'cooldown_seconds', cooldown_seconds,
  'max_retries', max_retries,
  'requeue', CASE WHEN requeue = 1 THEN json('true') ELSE json('false') END,
  'task_type', task_type,
  'args', args,
  'created_at', created_at,
  'updated_at', updated_at
)
FROM tasks
WHERE name = '%s';
`

func (db *DB) GetTask(name string) (*Task, error) {
	payload := fmt.Sprintf(getTaskFmt, name)

	if err := db.Query(payload); err != nil {
		return nil, err
	}

	resultBytes := []byte(db.GetResult())
	if len(resultBytes) == 0 {
		return nil, nil // no rows
	}

	var task Task
	if err := json.Unmarshal(resultBytes, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task JSON: %w", err)
	}

	return &task, nil
}

const listTasksFmt = `
SELECT json_group_array(
  json_object(
    'id', id,
    'name', name,
    'enabled', CASE WHEN enabled = 1 THEN json('true') ELSE json('false') END,
    'priority', priority,
    'cooldown_seconds', cooldown_seconds,
    'max_retries', max_retries,
    'requeue', CASE WHEN requeue = 1 THEN json('true') ELSE json('false') END,
    'task_type', task_type,
    'args', args,
    'created_at', created_at,
    'updated_at', updated_at
  )
)
FROM tasks
ORDER BY priority ASC, name ASC;
`

func (db *DB) ListTasks() ([]Task, error) {
	alfredo.VerbosePrintln("[db] Listing all tasks (begin)")
	defer alfredo.VerbosePrintln("[db] Listing all tasks (end)")

	if err := db.Query(listTasksFmt); err != nil {
		return nil, err
	}

	result := db.GetResult()
	if result == "" {
		return []Task{}, nil
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks JSON: %w", err)
	}

	alfredo.VerbosePrintf("[db] Loaded %d tasks", len(tasks))
	return tasks, nil
}

const enableTaskFmt = `
UPDATE tasks SET enabled = %d, updated_at = {{now}} 
WHERE name = '%s';
SELECT changes();
`

func (db *DB) EnableTask(name string, enabled bool) error {
	var enabledInt int
	switch enabled {
	case true:
		enabledInt = 1
	case false:
		enabledInt = 0
	}
	payload := fmt.Sprintf(enableTaskFmt,
		enabledInt, // bool → 1/0 via fmt.Sprintf %d
		name,
	)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	if rowsAffected != 1 {
		return fmt.Errorf("enable/disable affected %d rows (expected 1)", rowsAffected)
	}

	return nil
}

const deleteTaskFmt = `
DELETE FROM task_locks WHERE task_id NOT IN (SELECT task_id FROM tasks);
DELETE FROM tasks WHERE name = '%s';
SELECT changes();
`

func (db *DB) DeleteTask(name string) error {
	payload := fmt.Sprintf(deleteTaskFmt, name)

	if err := db.Query(payload); err != nil {
		return err
	}

	rowsAffected := db.GetResultInt64()
	if rowsAffected != 1 {
		return fmt.Errorf("delete affected %d rows (expected 1)", rowsAffected)
	}

	return nil
}

// RefreshTask clears execution history for a task so it can run again
// This is useful for one-shot tasks that need to be re-run
func (db *DB) RefreshTask(name string) error {
	// First verify the task exists and get its ID
	query := fmt.Sprintf("SELECT id FROM tasks WHERE name = '%s';",
		strings.ReplaceAll(name, "'", "''")) // Escape single quotes

	if err := db.Query(query); err != nil {
		return err
	}

	taskID := db.GetResultInt64()
	if taskID == 0 {
		return fmt.Errorf("task not found: %s", name)
	}

	// Delete all execution records for this task
	query = fmt.Sprintf("DELETE FROM task_executions WHERE task_id = %d;", taskID)
	if err := db.Query(query); err != nil {
		return fmt.Errorf("failed to clear executions: %w", err)
	}

	// Delete all metrics for this task
	query = fmt.Sprintf("DELETE FROM task_metrics WHERE task_id = %d;", taskID)
	if err := db.Query(query); err != nil {
		return fmt.Errorf("failed to clear metrics: %w", err)
	}

	// Clear any locks
	query = fmt.Sprintf("DELETE FROM task_locks WHERE task_id = %d;", taskID)
	if err := db.Query(query); err != nil {
		return fmt.Errorf("failed to clear locks: %w", err)
	}

	return nil
}
