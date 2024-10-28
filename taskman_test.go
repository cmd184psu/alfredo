package alfredo

import (
	"testing"
)

func mockFunction(file string) bool {
	if file == "success.txt" {
		return true
	}
	return false
}

// TestConcurrent tests the Concurrent function with a mock callback
func TestConcurrent(t *testing.T) {
	x := 5                      // Number of concurrent executions
	remoteFile := "success.txt" // Test case for successful result

	// Run Concurrent with the mock function and remoteFile
	results := Concurrent(mockFunction, remoteFile, x)

	// Check if all results are true, as expected for "success.txt"
	for i, result := range results {
		if result != true {
			t.Errorf("Test failed at index %d: expected true, got false", i)
		}
	}

	// Test case for a file that should fail
	remoteFile = "failure.txt"
	results = Concurrent(mockFunction, remoteFile, x)

	// Check if all results are false, as expected for "failure.txt"
	for i, result := range results {
		if result != false {
			t.Errorf("Test failed at index %d: expected false, got true", i)
		}
	}
}
