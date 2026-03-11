package ctq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cmd184psu/alfredo"
)

// Coordinator manages the task queue and provides API endpoints
type Coordinator struct {
	db         *DB
	httpAddr   string
	httpServer *http.Server
}

func NewCoordinator(db *DB, httpAddr string) *Coordinator {
	return &Coordinator{
		db:       db,
		httpAddr: httpAddr,
	}
}

// Start begins the coordinator service
func (c *Coordinator) Start() error {
	fmt.Printf("[coordinator] Starting on %q...\n", c.httpAddr)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Task management endpoints
	mux.HandleFunc("/tasks", c.handleTasks)
	mux.HandleFunc("/tasks/add", c.handleAddTask)
	mux.HandleFunc("/tasks/enable", c.handleEnableTask)
	mux.HandleFunc("/tasks/disable", c.handleDisableTask)
	mux.HandleFunc("/tasks/delete", c.handleDeleteTask)
	mux.HandleFunc("/tasks/refresh", c.handleRefreshTask)

	// Queue management endpoints
	mux.HandleFunc("/queue/pause", c.handlePauseQueue)
	mux.HandleFunc("/queue/resume", c.handleResumeQueue)
	mux.HandleFunc("/queue/status", c.handleQueueStatus)

	// Observability endpoints
	mux.HandleFunc("/metrics", c.handleMetrics)
	mux.HandleFunc("/executions", c.handleExecutions)
	mux.HandleFunc("/health", c.handleHealth)

	c.httpServer = &http.Server{
		Addr:    alfredo.StripProtocol(c.httpAddr),
		Handler: mux,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("[coordinator] Received shutdown signal\n")
		c.httpServer.Close()
	}()

	// Start server
	if err := c.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// handleTasks lists all tasks
func (c *Coordinator) handleTasks(w http.ResponseWriter, r *http.Request) {
	alfredo.VerbosePrintln("[coordinator] Begin HandleTasks")
	defer alfredo.VerbosePrintln("[coordinator] End HandleTasks")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tasks, err := c.db.ListTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleAddTask adds or updates a task
func (c *Coordinator) handleAddTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.db.AddTask(&task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleEnableTask enables a task
func (c *Coordinator) handleEnableTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	if err := c.db.EnableTask(name, true); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDisableTask disables a task
func (c *Coordinator) handleDisableTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	if err := c.db.EnableTask(name, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteTask deletes a task
func (c *Coordinator) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	if err := c.db.DeleteTask(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleRefreshTask clears execution history so a one-shot task can run again
func (c *Coordinator) handleRefreshTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}
	if err := c.db.RefreshTask(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[coordinator] Task '%s' refreshed\n", name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refreshed"})
}

// handlePauseQueue pauses the queue
func (c *Coordinator) handlePauseQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pausedBy := r.URL.Query().Get("by")
	if pausedBy == "" {
		pausedBy = "coordinator"
	}

	if err := c.db.SetQueuePaused(true, pausedBy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[coordinator] Queue paused by %s\n", pausedBy)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// handleResumeQueue resumes the queue
func (c *Coordinator) handleResumeQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := c.db.SetQueuePaused(false, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[coordinator] Queue resumed\n")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "running"})
}

const getQueueStatusFmt = `
SELECT paused, paused_at, paused_by 
FROM queue_state 
WHERE id = 1;
`

func (c *Coordinator) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := c.db.Query(getQueueStatusFmt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := c.db.GetResult()
	if result == "" {
		http.Error(w, "queue state not found", http.StatusNotFound)
		return
	}

	// Parse pipe-delimited: "1|2026-02-07 23:16:48|user123"
	fields := strings.Split(strings.TrimSpace(result), "|")
	if len(fields) < 1 {
		http.Error(w, "invalid queue status format", http.StatusInternalServerError)
		return
	}

	paused, err := strconv.ParseBool(fields[0]) // "1" or "0"
	if err != nil {
		http.Error(w, "invalid paused field", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"paused": paused,
	}

	// Optional: Parse paused_at (fields[1]), paused_by (fields[2])
	if paused && len(fields) > 1 && fields[1] != "" {
		layout := "2006-01-02 15:04:05"
		if pausedAt, err := time.Parse(layout, fields[1]); err == nil {
			resp["paused_at"] = pausedAt
		}
		if len(fields) > 2 && fields[2] != "" {
			resp["paused_by"] = fields[2]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

const metricsQueryFmt = `
SELECT json_group_array(value)
FROM (
  SELECT json_object(
    'task_name',      t.name,
    'success_count',  COUNT(CASE WHEN tm.status = 'success' THEN 1 END),
    'failed_count',   COUNT(CASE WHEN tm.status = 'failed' THEN 1 END),
    'avg_duration_ms',COALESCE(AVG(CASE WHEN tm.status = 'success' THEN tm.duration_ms END), NULL),
    'min_duration_ms',COALESCE(MIN(tm.duration_ms), NULL),
    'max_duration_ms',COALESCE(MAX(tm.duration_ms), NULL),
'last_execution', CASE WHEN MAX(tm.recorded_at) IS NULL THEN NULL ELSE %d END  ) as value
  FROM tasks t
  LEFT JOIN task_metrics tm ON t.id = tm.task_id
      AND tm.recorded_at < %d
  WHERE 1=1 %s
  GROUP BY t.id, t.name
) sub
ORDER BY json_extract(value, '$.task_name');
`

func (c *Coordinator) handleMetrics(w http.ResponseWriter, r *http.Request) {
	log.Println("[coordinator] Begin HandleMetrics")
	defer log.Println("[coordinator] End HandleMetrics")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskName := r.URL.Query().Get("task")
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil {
			hours = h
		}
	}

	whereClause := ""
	if taskName != "" {
		whereClause = fmt.Sprintf("AND t.name = '%s'", taskName)
	}

	nowMs := time.Now().UnixMilli()
	cutoffMs := nowMs - int64(hours*3600*1000) // hours → milliseconds

	payload := fmt.Sprintf(metricsQueryFmt, nowMs, cutoffMs, whereClause)
	//	payload := fmt.Sprintf(metricsQueryFmt, hours, whereClause)

	fmt.Println("=================")
	fmt.Println(payload)
	fmt.Println("=================")

	if err := c.db.Query(payload); err != nil { //panics here
		//panic("failed to query metrics: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// fmt.Println("GetResult: (raw string)")
	// fmt.Println(c.db.GetResult())
	// fmt.Println("=================")

	resultBytes := []byte(c.db.GetResult())
	if len(resultBytes) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	type Metric struct {
		TaskName      string     `json:"task_name"`
		SuccessCount  int        `json:"success_count"`
		FailedCount   int        `json:"failed_count"`
		AvgDurationMs *float64   `json:"avg_duration_ms"`
		MinDurationMs *int64     `json:"min_duration_ms"`
		MaxDurationMs *int64     `json:"max_duration_ms"`
		LastExecution *time.Time `json:"last_execution"`
	}

	var metrics []Metric
	if err := json.Unmarshal([]byte(c.db.GetResult()), &metrics); err != nil {
		panic(fmt.Sprintf("failed to parse metrics: %v", err))

	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

const executionsQueryFmt = `
SELECT json_group_array(
  json_object(
    'id',           te.id,
    'task_id',      te.task_id,
    'task_name',    t.name,
    'started_at',   te.started_at,
    'finished_at',  te.finished_at,
    'status',       te.status,
    'error_message',te.error_message,
    'retry_count',  te.retry_count,
    'worker_id',    te.worker_id,
    'duration_ms',  te.duration_ms
  )
)
FROM task_executions te
JOIN tasks t ON te.task_id = t.id
WHERE 1=1 %s
ORDER BY te.started_at DESC
LIMIT %d;
`

func (c *Coordinator) handleExecutions(w http.ResponseWriter, r *http.Request) {
	log.Println("[coordinator] Begin HandleExecutions")
	defer log.Println("[coordinator] End HandleExecutions")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	taskName := r.URL.Query().Get("task")
	whereClause := ""
	if taskName != "" {
		whereClause = fmt.Sprintf("AND t.name = '%s'", taskName)
	}

	payload := fmt.Sprintf(executionsQueryFmt, whereClause, limit)

	if err := c.db.Query(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	alfredo.VerbosePrintln("=================")
	alfredo.VerbosePrintln(payload)
	alfredo.VerbosePrintln("=================")
	alfredo.VerbosePrintln(c.db.GetResult())
	alfredo.VerbosePrintln("=================")

	type ExecutionDetail struct {
		ID           int64      `json:"id"`
		TaskID       int64      `json:"task_id"`
		TaskName     string     `json:"task_name"`
		StartedAt    *time.Time `json:"started_at"`
		FinishedAt   *time.Time `json:"finished_at"`
		Status       string     `json:"status"`
		ErrorMessage *string    `json:"error_message"`
		RetryCount   int        `json:"retry_count"`
		WorkerID     *string    `json:"worker_id"`
		DurationMs   *int64     `json:"duration_ms"`
	}

	var executions []ExecutionDetail

	rows := c.db.GetResultAsSlice()

	if len(rows) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	result := c.db.GetResult()
	if err := json.Unmarshal([]byte(result), &executions); err != nil {
		panic(fmt.Sprintf("failed to parse executionDetail error: %v", err))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func (c *Coordinator) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check database connectivity with simple query
	const healthCheckFmt = "SELECT 1;"
	if err := c.db.Query(healthCheckFmt); err != nil {
		http.Error(w, "Database unhealthy: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
