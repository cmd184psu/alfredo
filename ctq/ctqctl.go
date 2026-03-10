package ctq

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cmd184psu/alfredo"
)

const DefaultCoordinatorURL = "http://127.0.0.1:4444"

func RunCLI() {
	var coordinatorURL string
	flag.StringVar(&coordinatorURL, "url", DefaultCoordinatorURL, "Coordinator URL")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	subArgs := args[1:]

	switch command {
	case "version":
		fmt.Printf(ctq_version_fmt, alfredo.BuildVersion())
		if alfredo.FileExistsEasy("/opt/cloudian-bucket-tools/VERSION") {
			sl, _ := alfredo.ReadFileToSlice("/opt/cloudian-bucket-tools/VERSION", true)
			fmt.Printf("Cloudian Bucket Tools Version: %s\n", sl[0])
		} else {
			fmt.Println("Cloudian Bucket Tools Version: unknown")
		}
		os.Exit(0)

	case "add":
		handleAdd(coordinatorURL)
	case "list":
		handleList(coordinatorURL)
	case "enable":
		handleEnable(coordinatorURL, true)
	case "disable":
		handleEnable(coordinatorURL, false)
	case "delete":
		handleDelete(coordinatorURL, subArgs)
	case "refresh":
		handleRefresh(coordinatorURL)
	case "pause":
		handlePause(coordinatorURL)
	case "resume":
		handleResume(coordinatorURL)
	case "status":
		handleStatus(coordinatorURL)
	case "metrics":
		handleMetrics(coordinatorURL)
	case "executions":
		handleExecutions(coordinatorURL)
	case "health":
		handleHealth(coordinatorURL)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ctqctl [options] <command>

Commands:
  add         Add a task (reads JSON from stdin or -file)
  list        List all tasks
  enable      Enable a task (-name required)
  disable     Disable a task (-name required)
  delete      Delete a task (-name required)
  refresh     Clear execution history so task can run again (-name required)
  pause       Pause the queue
  resume      Resume the queue
  status      Show queue status
  metrics     Show task metrics (-task optional, -hours optional)
  executions  Show recent executions (-task optional, -limit optional)
  health      Check coordinator health

Options:
  -url string
        Coordinator URL (default "http://127.0.0.1:4444")

Examples:
  # Add a task
  ctqctl add <<EOF
  {
    "name": "backup",
    "enabled": true,
    "priority": 50,
    "cooldown_seconds": 3600,
    "max_retries": 3,
    "requeue": true,
    "task_type": "shell",
    "args": "{\"shell\": \"tar -czf /backup/data.tar.gz /data\"}"
  }
  EOF

  # List tasks
  ctqctl list

  # Pause queue
  ctqctl pause

  # Refresh a one-shot task to run it again
  ctqctl refresh -name migrate-v2

  # View metrics
  ctqctl metrics -task backup -hours 24
`)
}

func handleAdd(baseURL string) {
	var taskJSON []byte
	var err error

	// Read from stdin
	taskJSON, err = io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Validate JSON
	var task map[string]interface{}
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid JSON: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post(baseURL+"/tasks/add", "application/json", bytes.NewReader(taskJSON))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Println("Task added successfully")
}

func handleList(baseURL string) {
	resp, err := http.Get(baseURL + "/tasks")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Optional: Log body without consuming for decoder
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading body: %v\n", err)
		os.Exit(1)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body)) // Reset for decoder

	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response (1): %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tENABLED\tPRIORITY\tCOOLDOWN\tRETRIES\tREQUEUE\tTYPE")
	for _, task := range tasks {
		fmt.Fprintf(w, "%d\t%s\t%v\t%d\t%ds\t%d\t%v\t%s\n",
			task.ID, task.Name, task.Enabled, task.Priority,
			task.CooldownSeconds, task.MaxRetries, task.Requeue, task.TaskType)
	}
	w.Flush()
}

func handleEnable(baseURL string, enable bool) {
	var name string
	flag.StringVar(&name, "name", "", "Task name")
	flag.CommandLine.Parse(os.Args[2:])

	if name == "" {
		fmt.Fprintf(os.Stderr, "Error: -name is required\n")
		os.Exit(1)
	}

	endpoint := "/tasks/enable"
	if !enable {
		endpoint = "/tasks/disable"
	}

	resp, err := http.Post(baseURL+endpoint+"?name="+name, "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	fmt.Printf("Task '%s' %s\n", name, action)
}

func handleDelete(baseURL string, subArgs []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)

	var name string
	fs.StringVar(&name, "name", "", "Task name")

	// Parse the subcommand-specific args: "-name backup"
	if err := fs.Parse(subArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// flag.StringVar(&name, "name", "", "Task name")
	// flag.CommandLine.Parse(os.Args[2:])

	if name == "" {
		fmt.Fprintf(os.Stderr, "Error: -name is required\n")
		os.Exit(1)
	}

	req, err := http.NewRequest(http.MethodDelete, baseURL+"/tasks/delete?name="+name, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Printf("Task '%s' deleted\n", name)
}

func handleRefresh(baseURL string) {
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	name := fs.String("name", "", "Task name")
	fs.Parse(os.Args[2:])

	if *name == "" {
		fmt.Fprintf(os.Stderr, "Error: -name is required\n")
		os.Exit(1)
	}

	resp, err := http.Post(baseURL+"/tasks/refresh?name="+*name, "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}
	fmt.Printf("Task '%s' refreshed - execution history cleared, will run again\n", *name)
}

func handlePause(baseURL string) {
	resp, err := http.Post(baseURL+"/queue/pause", "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Println("Queue paused")
}

func handleResume(baseURL string) {
	resp, err := http.Post(baseURL+"/queue/resume", "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Println("Queue resumed")
}

func handleStatus(baseURL string) {
	resp, err := http.Get(baseURL + "/queue/status")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response(2): %v\n", err)
		os.Exit(1)
	}

	paused := status["paused"].(bool)
	if paused {
		fmt.Println("Queue Status: PAUSED")
		if pausedAt, ok := status["paused_at"].(string); ok {
			fmt.Printf("Paused At: %s\n", pausedAt)
		}
		if pausedBy, ok := status["paused_by"].(string); ok {
			fmt.Printf("Paused By: %s\n", pausedBy)
		}
	} else {
		fmt.Println("Queue Status: RUNNING")
	}
}

func handleMetrics(baseURL string) {
	var taskName string
	var hours int
	flag.StringVar(&taskName, "task", "", "Task name (optional)")
	flag.IntVar(&hours, "hours", 24, "Hours to look back")
	flag.CommandLine.Parse(os.Args[2:])

	url := fmt.Sprintf("%s/metrics?hours=%d", baseURL, hours)
	if taskName != "" {
		url += "&task=" + taskName
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var metrics []struct {
		TaskName      string     `json:"task_name"`
		SuccessCount  int        `json:"success_count"`
		FailedCount   int        `json:"failed_count"`
		AvgDurationMs *float64   `json:"avg_duration_ms"`
		MinDurationMs *int64     `json:"min_duration_ms"`
		MaxDurationMs *int64     `json:"max_duration_ms"`
		LastExecution *time.Time `json:"last_execution"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response(3): %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK\tSUCCESS\tFAILED\tAVG_MS\tMIN_MS\tMAX_MS\tLAST_EXEC")
	for _, m := range metrics {
		avgMs := "N/A"
		if m.AvgDurationMs != nil {
			avgMs = fmt.Sprintf("%.0f", *m.AvgDurationMs)
		}
		minMs := "N/A"
		if m.MinDurationMs != nil {
			minMs = fmt.Sprintf("%d", *m.MinDurationMs)
		}
		maxMs := "N/A"
		if m.MaxDurationMs != nil {
			maxMs = fmt.Sprintf("%d", *m.MaxDurationMs)
		}
		lastExec := "Never"
		if m.LastExecution != nil {
			lastExec = m.LastExecution.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\n",
			m.TaskName, m.SuccessCount, m.FailedCount, avgMs, minMs, maxMs, lastExec)
	}
	w.Flush()
}

func handleExecutions(baseURL string) {
	var taskName string
	var limit int
	flag.StringVar(&taskName, "task", "", "Task name (optional)")
	flag.IntVar(&limit, "limit", 50, "Number of executions to show")
	flag.CommandLine.Parse(os.Args[2:])

	url := fmt.Sprintf("%s/executions?limit=%d", baseURL, limit)
	if taskName != "" {
		url += "&task=" + taskName
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var executions []struct {
		ID           int64      `json:"id"`
		TaskName     string     `json:"task_name"`
		StartedAt    *time.Time `json:"started_at"`
		FinishedAt   *time.Time `json:"finished_at"`
		Status       string     `json:"status"`
		ErrorMessage *string    `json:"error_message"`
		RetryCount   int        `json:"retry_count"`
		DurationMs   *int64     `json:"duration_ms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&executions); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response(4): %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTASK\tSTATUS\tDURATION\tRETRIES\tFINISHED\tERROR")
	for _, e := range executions {
		duration := "N/A"
		if e.DurationMs != nil {
			duration = fmt.Sprintf("%dms", *e.DurationMs)
		}
		finished := "N/A"
		if e.FinishedAt != nil {
			finished = e.FinishedAt.Format("2006-01-02 15:04:05")
		}
		errMsg := ""
		if e.ErrorMessage != nil {
			errMsg = *e.ErrorMessage
			if len(errMsg) > 50 {
				errMsg = errMsg[:47] + "..."
			}
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
			e.ID, e.TaskName, e.Status, duration, e.RetryCount, finished, errMsg)
	}
	w.Flush()
}

func handleHealth(baseURL string) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Coordinator: HEALTHY")
	} else {
		fmt.Println("Coordinator: UNHEALTHY")
		os.Exit(1)
	}
}
