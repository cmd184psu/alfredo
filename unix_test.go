package alfredo

import (
	"fmt"
	"os"
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
