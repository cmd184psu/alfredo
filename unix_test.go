package alfredo

import (
	"fmt"
	"os"
	"reflect"
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
func TestCapturePids(t *testing.T) {
	type args struct {
		main string
		hint string
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "Single process",
			args: args{
				main: "bash",
				hint: "",
			},
			want:    []int{os.Getpid()},
			wantErr: false,
		},
		{
			name: "Multiple processes",
			args: args{
				main: "go",
				hint: "",
			},
			want:    []int{}, // This will vary depending on the environment
			wantErr: false,
		},
		{
			name: "Non-existent process",
			args: args{
				main: "nonexistentprocess",
				hint: "",
			},
			want:    []int{},
			wantErr: false,
		},
		{
			name: "With hint",
			args: args{
				main: "bash",
				hint: "bash",
			},
			want:    []int{os.Getpid()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CapturePids(tt.args.main, tt.args.hint)
			if (err != nil) != tt.wantErr {
				t.Errorf("CapturePids() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CapturePids() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestPopen3Grep(t *testing.T) {
	testCases := []struct {
		name          string
		cmd           string
		musthave      string
		mustnothave   string
		expectedLines []string
		expectError   bool
	}{
		{
			name:          "Basic grep",
			cmd:           "echo \"hello world\"",
			musthave:      "hello",
			mustnothave:   "",
			expectedLines: []string{"\"hello world\""},
			expectError:   false,
		},
		{
			name:          "Grep with mustnothave",
			cmd:           "echo 'hello world'",
			musthave:      "hello",
			mustnothave:   "world",
			expectedLines: []string{},
			expectError:   false,
		},
		{
			name:          "Grep with multiple lines",
			cmd:           "echo -e 'hello world\nfoo bar'",
			musthave:      "foo",
			mustnothave:   "",
			expectedLines: []string{"foo bar"},
			expectError:   false,
		},
		{
			name:          "Grep with musthave and mustnothave",
			cmd:           "echo -e 'hello world\nfoo bar'",
			musthave:      "foo",
			mustnothave:   "bar",
			expectedLines: []string{},
			expectError:   false,
		},
		{
			name:          "Non-existent command",
			cmd:           "nonexistentcommand",
			musthave:      "",
			mustnothave:   "",
			expectedLines: []string{},
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lines, err := Popen3Grep2(tc.cmd, tc.musthave, tc.mustnothave)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if !reflect.DeepEqual(lines, tc.expectedLines) {
				t.Errorf("Expected lines: %v, got: %v", tc.expectedLines, lines)
			}
		})
	}
}
func TestGetProcessList(t *testing.T) {
	testCases := []struct {
		name          string
		mainHint      string
		expectedError bool
	}{
		{
			name:          "Existing process",
			mainHint:      "go",
			expectedError: false,
		},
		{
			name:          "Non-existent process",
			mainHint:      "nonexistentprocess",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetProcessList(tc.mainHint)
			if (err != nil) != tc.expectedError {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}
			if tc.expectedError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectedError && len(result) == 0 && tc.mainHint != "nonexistentprocess" {
				t.Errorf("Expected some processes but got none")
			}
		})
	}
}

// func TestCapturePids(t *testing.T) {
// 	type args struct {
// 		main string
// 		hint string
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    []int
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := CapturePids(tt.args.main, tt.args.hint)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("CapturePids() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("CapturePids() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestWriteStringToFile(t *testing.T) {
	type args struct {
		filename string
		content  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Write to existing file",
			args: args{
				filename: "/tmp/testfile.txt",
				content:  "Hello, World!",
			},
			wantErr: false,
		},
		{
			name: "Write to non-existent directory",
			args: args{
				filename: "/tmp/nonexistentdir/testfile.txt",
				content:  "Hello, World!",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteStringToFile(tt.args.filename, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("WriteStringToFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// Verify file content
				data, err := os.ReadFile(tt.args.filename)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
				}
				if string(data) != tt.args.content {
					t.Errorf("File content mismatch. Expected: %s, Got: %s", tt.args.content, string(data))
				}
				// Clean up
				os.Remove(tt.args.filename)
			}
		})
	}
}
