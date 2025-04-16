// Integration test for MemoryManager
package alfredo

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func TestMemoryManagerIntegration(t *testing.T) {
	// Create a custom logger to capture output
	logFile, err := os.CreateTemp("", "memory_manager_test.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())

	fmt.Println("writing to log file: ", logFile.Name())
	logger := log.New(logFile, "", log.LstdFlags)
	options := DefaultMMOptions().WithInterval(1 * time.Second).WithForceGC(true).WithMemoryThreshold(1024 * 1024).WithLogger(logger)
	// Create and start the memory manager
	mm := NewMemoryManager(*options)
	mm.Start()

	// Run the memory manager for a short duration
	time.Sleep(5 * time.Second)

	// Stop the memory manager
	mm.Stop()

	fmt.Println("after stopping the memory manager")

	// Verify the log file contains expected output
	logFile.Seek(0, 0)
	logContent := make([]byte, 1024)
	n, _ := logFile.Read(logContent)
	if n == 0 {
		t.Errorf("Expected log output, but log file is empty")
	}
	fmt.Println("after log seek crap")

	// Optionally, check for specific log messages
	logOutput := string(logContent[:n])
	if !contains(logOutput, "Memory stats:") {
		t.Errorf("Expected memory stats in log output, but not found")
	}
	if !contains(logOutput, "Forced garbage collection") {
		t.Errorf("Expected forced garbage collection log, but not found")
	}
	if !contains(logOutput, "WARNING: Memory usage exceeds threshold") {
		t.Errorf("Expected memory threshold warning, but not found")
	}
	fmt.Println("the end of the test")
}

// Helper function to check if a substring exists in a string
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (len(substr) == 0 || len(str) >= len(substr) && (str[:len(substr)] == substr || contains(str[1:], substr)))
}
