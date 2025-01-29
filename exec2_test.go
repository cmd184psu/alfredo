package alfredo

import (
	"fmt"
	"testing"
	"time"
)

func TestCLIExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		args           []string
		requestPayload string
		sshHost        string
		sshKey         string
		timeout        time.Duration
		showSpinny     bool
		captureStdout  bool
		captureStderr  bool
		wantOutput     string
		wantExitCode   int
		wantErr        bool
	}{
		{
			name:          "Local command with stdout capture",
			command:       "echo hello",
			captureStdout: true,
			wantOutput:    "hello",
			wantExitCode:  0,
			wantErr:       false,
		},
		{
			name:          "Local command with stderr capture",
			command:       "sh",
			args:          []string{"-c", "echo error >&2"},
			captureStderr: true,
			wantOutput:    "error",
			wantExitCode:  0,
			wantErr:       false,
		},
		{
			name:         "Non-existent command",
			command:      "nonexistentcommand",
			wantOutput:   "",
			wantExitCode: -1,
			wantErr:      true,
		},
		{
			name:           "Local command with request payload",
			command:        "cat",
			requestPayload: "hello",
			captureStdout:  true,
			wantOutput:     "hello",
			wantExitCode:   0,
			wantErr:        false,
		},
		{
			name:         "Command with timeout",
			command:      "sleep",
			args:         []string{"10"},
			timeout:      1 * time.Second,
			wantOutput:   "",
			wantExitCode: -1,
			wantErr:      true,
		},
		{
			name:           "local md5sum",
			command:        "md5sum",
			requestPayload: "5555",
			captureStdout:  true,
			wantOutput:     "6074c6aa3488f3c2dddff2a7ca821aab  -",
			wantExitCode:   0,
			wantErr:        false,
		},
		{
			name:           "remote md5sum",
			sshHost:        "localhost",
			sshKey:         ExpandTilde("~/.ssh/id_rsa"),
			command:        "/usr/local/bin/md5sum",
			captureStdout:  true,
			requestPayload: "5555",
			wantOutput:     "6074c6aa3488f3c2dddff2a7ca821aab  -",
			wantExitCode:   0,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewCLIExecutor().
				WithCommand(tt.command, tt.args...).
				WithRequestPayload(tt.requestPayload).
				WithSSH(tt.sshHost, tt.sshKey).
				WithTimeout(tt.timeout).
				WithSpinny(tt.showSpinny).
				WithCaptureStdout(tt.captureStdout).
				WithCaptureStderr(tt.captureStderr).
				WithTrimWhiteSpace(true)

			err := executor.Execute()
			fmt.Println("output: ", executor.GetResponseBody())
			if (err != nil) != tt.wantErr {
				t.Errorf("CLIExecutor.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotExitCode := executor.GetStatusCode()
			gotOutput := executor.GetResponseBody()
			fmt.Println("exit code: ", executor.GetStatusCode())
			if gotOutput != tt.wantOutput {
				t.Errorf("CLIExecutor.Execute() gotOutput = %q, want %q", gotOutput, tt.wantOutput)
			}
			if gotExitCode != tt.wantExitCode {
				t.Errorf("CLIExecutor.Execute() gotExitCode = %v, want %v", gotExitCode, tt.wantExitCode)
			}
		})
	}
}
