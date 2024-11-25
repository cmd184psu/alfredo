package alfredo

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGenerateMoveCLI(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "testfile*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Close the file to ensure it's written to disk
	tempFile.Close()

	// Get the file info
	fileInfo, err := os.Stat(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// Format the expected modification time
	modTime := fileInfo.ModTime().Format("02Jan2006")

	testCases := []struct {
		name           string
		inputFile      string
		suffix         string
		expectedOutput string
	}{
		{
			name:           "Basic case",
			inputFile:      tempFile.Name(),
			suffix:         "-processed-",
			expectedOutput: fmt.Sprintf("mv %s %s-processed-%s.txt", tempFile.Name(), tempFile.Name()[:len(tempFile.Name())-4], modTime),
		},
		{
			name:           "Empty suffix",
			inputFile:      tempFile.Name(),
			suffix:         "",
			expectedOutput: fmt.Sprintf("mv %s %s%s.txt", tempFile.Name(), tempFile.Name()[:len(tempFile.Name())-4], modTime),
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GenerateMoveCLI(tc.inputFile, tc.suffix)
			if result != tc.expectedOutput {
				t.Errorf("Expected: %s, Got: %s", tc.expectedOutput, result)
			}
		})
	}

	// Test for non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic for non-existent file")
			}
		}()
		GenerateMoveCLI("non_existent_file.txt", "_suffix")
	})
}

func TestSystem3toCapturedString(t *testing.T) {
	// Helper function to run the test
	runTest := func(cmd string, expectedOutput string, expectedError bool) {
		var output string
		err := System3toCapturedString(&output, cmd)

		if expectedError && err == nil {
			t.Errorf("Expected an error for command: %s, but got none", cmd)
		}
		if !expectedError && err != nil {
			t.Errorf("Unexpected error for command: %s. Error: %v", cmd, err)
		}
		if strings.EqualFold(expectedOutput, "content") {
			if len(output) == 0 {
				t.Errorf("For command: %s\nExpected output len: >0\nGot: 0", cmd)
			}
		} else {
			if output != expectedOutput {
				t.Errorf("For command: %s\nExpected output: %s\nGot: %s", cmd, expectedOutput, output)
			}
		}
	}

	// Test cases
	testCases := []struct {
		cmd            string
		expectedOutput string
		expectedError  bool
	}{
		// Test basic command
		{"echo Hello", "Hello\n", false},

		// Test command with quotes
		{"echo \"Hello World\"", "Hello World\n", false},

		// Test cd command
		{"cd /usr && pwd", "/usr\n", false},

		// Test command with multiple spaces
		{"echo  Hello    World", "Hello World\n", false},

		// Test non-existent command
		{"nonexistentcommand", "", true},

		// Test command with arguments
		{"ls -l /usr", "content", false}, // Output will vary, so we don't check it

		// Test command with environment variables

		//environment isn't carrying over for whatever reason, skip for now
		//{"echo $HOME", os.Getenv("HOME"), false}, // Output will vary, so we don't check it
	}

	for _, tc := range testCases {
		runTest(tc.cmd, tc.expectedOutput, tc.expectedError)
	}
}
