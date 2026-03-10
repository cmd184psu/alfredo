package ctq

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cmd184psu/alfredo"
)

// TaskExecutor executes tasks based on their type
type TaskExecutor struct {
	db       *DB
	workerID string
}

func NewTaskExecutor(db *DB, workerID string) *TaskExecutor {
	return &TaskExecutor{
		db:       db,
		workerID: workerID,
	}
}

// Execute runs a task and records the results
func (te *TaskExecutor) Execute(twe *TaskWithExecution) error {
	task := twe.Task

	// Determine retry count
	retryCount := 0
	if twe.LastExecution != nil && twe.LastExecution.Status == "failed" {
		retryCount = twe.LastExecution.RetryCount
	}

	fmt.Printf("[%s] Executing task: %s (attempt %d/%d)\n",
		te.workerID, task.Name, retryCount+1, task.MaxRetries+1)

	// Create execution record
	executionID, err := te.db.CreateExecution(task.ID, te.workerID)
	if err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	startTime := time.Now()

	// Execute the task based on type
	var execErr error
	switch task.TaskType {
	case "exec":
		execErr = te.executeCommand(task)
	case "script":
		execErr = te.executeScript(task)
	case "shell":
		execErr = te.executeShell(task)
	default:
		execErr = fmt.Errorf("unknown task type: %s", task.TaskType)
	}

	duration := time.Since(startTime)
	durationMs := duration.Milliseconds()

	// Update execution record
	status := "success"
	var errorMsg *string
	if execErr != nil {
		status = "failed"
		errStr := execErr.Error()
		errorMsg = &errStr
		fmt.Printf("[%s] Task %s failed: %v (duration: %v)\n",
			te.workerID, task.Name, execErr, duration)
	} else {
		fmt.Printf("[%s] Task %s completed successfully (duration: %v)\n",
			te.workerID, task.Name, duration)
	}

	if err := te.db.UpdateExecution(executionID, status, errorMsg, durationMs); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Record metric
	if err := te.db.RecordMetric(task.ID, durationMs, status); err != nil {
		fmt.Printf("[%s] Warning: failed to record metric: %v\n", te.workerID, err)
	}

	return execErr
}

// executeCommand executes a command with arguments
func (te *TaskExecutor) executeCommand(task Task) error {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(task.Args), &args); err != nil {
		return fmt.Errorf("invalid args JSON: %w", err)
	}

	command, ok := args["command"].(string)
	if !ok {
		return fmt.Errorf("missing 'command' in args")
	}

	// Parse command arguments
	var cmdArgs []string
	if argsIface, ok := args["args"]; ok {
		switch v := argsIface.(type) {
		case []interface{}:
			for _, arg := range v {
				if s, ok := arg.(string); ok {
					cmdArgs = append(cmdArgs, s)
				}
			}
		case string:
			// Split string args
			if v != "" {
				cmdArgs = strings.Fields(v)
			}
		}
	}

	// Set working directory if specified
	workDir := "/"
	if wd, ok := args["workdir"].(string); ok {
		workDir = wd
	}

	// Set environment variables if specified
	env := os.Environ()
	if envMap, ok := args["env"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if s, ok := v.(string); ok {
				env = append(env, fmt.Sprintf("%s=%s", k, s))
			}
		}
	}

	cmd := exec.Command(command, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// executeScript executes a script file
func (te *TaskExecutor) executeScript(task Task) error {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(task.Args), &args); err != nil {
		return fmt.Errorf("invalid args JSON: %w", err)
	}

	scriptPath, ok := args["path"].(string)
	if !ok {
		return fmt.Errorf("missing 'path' in args")
	}

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script not found: %w", err)
	}

	// Parse script arguments
	var scriptArgs []string
	if argsIface, ok := args["args"]; ok {
		switch v := argsIface.(type) {
		case []interface{}:
			for _, arg := range v {
				if s, ok := arg.(string); ok {
					scriptArgs = append(scriptArgs, s)
				}
			}
		}
	}

	// Set working directory if specified
	workDir := "/"
	if wd, ok := args["workdir"].(string); ok {
		workDir = wd
	}

	cmd := exec.Command(scriptPath, scriptArgs...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// executeShell executes a shell command
func (te *TaskExecutor) executeShell(task Task) error {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(task.Args), &args); err != nil {
		return fmt.Errorf("invalid args JSON: %w", err)
	}

	shellCmd, ok := args["shell"].(string)
	if !ok {
		return fmt.Errorf("missing 'shell' in args")
	}

	// DEBUG: Log what we're about to execute
	fmt.Printf("[DEBUG] Shell command: %s\n", shellCmd)

	workDir := "/"
	if wd, ok := args["workdir"].(string); ok {
		workDir = wd
	}

	fmt.Printf("[DEBUG] Working directory: %s\n", workDir)

	exe := alfredo.NewCLIExecutor().AsLongRunning().DumpOutput().WithSudo(true).WithCommand(shellCmd)

	return exe.Execute()

}
