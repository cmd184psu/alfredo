package alfredo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type CLIExecutor struct {
	command        string
	args           []string
	requestPayload string
	sshHost        string
	sshKey         string
	timeout        time.Duration
	showSpinny     bool
	captureStdout  bool
	captureStderr  bool
	statusCode     int
	responseBody   string
	trimWhiteSpace bool
	directory      string
}

const default_timeout = 5 * time.Second

func NewCLIExecutor() *CLIExecutor {
	ex := &CLIExecutor{}
	t := default_timeout
	//ex.SetTimeout(t)
	return ex.WithTimeout(t)
}

func (c *CLIExecutor) WithCommand(command string, args ...string) *CLIExecutor {
	c.command = command
	c.args = args
	return c
}

func (c *CLIExecutor) WithRequestPayload(payload string) *CLIExecutor {
	c.requestPayload = payload
	return c
}

func (c *CLIExecutor) WithSSH(host, key string) *CLIExecutor {
	c.sshHost = host
	c.sshKey = key
	return c
}

func (c *CLIExecutor) WithDirectory(directory string) *CLIExecutor {
	c.directory = directory
	return c
}
func (c *CLIExecutor) WithTimeout(timeout time.Duration) *CLIExecutor {
	if timeout == 0 {
		return c
	}
	c.timeout = timeout
	return c
}

// stick around for 100 days
func (c *CLIExecutor) AsLongRunning() *CLIExecutor {
	return c.WithTimeout(8640000 * time.Second)
}

func (c *CLIExecutor) WithSpinny(show bool) *CLIExecutor {
	c.showSpinny = show
	return c
}

func (c *CLIExecutor) WithCaptureStdout(capture bool) *CLIExecutor {
	c.captureStdout = capture
	return c
}

func (c *CLIExecutor) WithCaptureStderr(capture bool) *CLIExecutor {
	c.captureStderr = capture
	return c
}

func (c *CLIExecutor) WithResponseBody(responseBody string) *CLIExecutor {
	c.responseBody = responseBody
	return c
}

func (c *CLIExecutor) GetResponseBody() string {
	if c.trimWhiteSpace {
		return strings.TrimSpace(c.responseBody)
	}
	return c.responseBody
}

func (c *CLIExecutor) WithTrimWhiteSpace(trim bool) *CLIExecutor {
	c.trimWhiteSpace = trim
	return c
}

func (c *CLIExecutor) GetTrimWhiteSpace() bool {
	return c.trimWhiteSpace
}

func (c *CLIExecutor) WithStatusCode(statusCode int) *CLIExecutor {
	c.statusCode = statusCode
	return c
}

func (c *CLIExecutor) GetStatusCode() int {
	return c.statusCode
}

func (c *CLIExecutor) GetCli() string {
	if c.sshHost != "" {
		return fmt.Sprintf("ssh -i %s %s \"%s %s\"", c.sshKey, c.sshHost, c.command, strings.Join(c.args, " "))
	}
	return fmt.Sprintf("%s %s", c.command, strings.Join(c.args, " "))
}

func (c *CLIExecutor) Execute() error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", c.GetCli())
		return nil
	}
	var cmd *exec.Cmd

	if len(c.args) == 0 && len(c.command) == 0 {
		return fmt.Errorf("no command provided")
	}
	if len(c.directory) > 0 {
		if len(c.sshHost) > 0 {
			c.command = "cd " + c.directory + " && " + c.command
		}
	}
	if len(c.args) == 0 && strings.Contains(c.command, " ") {
		parts := strings.Split(c.command, " ")
		c.command = parts[0]
		c.args = parts[1:]
	}
	cwd := EatErrorReturnString(os.Getwd())
	VerbosePrintf("cwd: %s\n", cwd)
	if len(c.directory) > 0 {
		VerbosePrintf("want to chdir to %s\n", c.directory)
	} else {
		VerbosePrintf("don't change directories\n")
	}
	if c.sshHost != "" {
		cmd = exec.Command("ssh", "-i", c.sshKey, c.sshHost, c.command, strings.Join(c.args, " "))
	} else {
		VerbosePrintf("command: %s %s\n", c.command, strings.Join(c.args, " "))
		cmd = exec.Command(c.command, c.args...)
		if len(c.directory) > 0 {
			cmd.Dir = c.directory

			// if err := os.Chdir(c.directory); err != nil {
			// 	panic(err.Error())
			// }

			//			VerbosePrintf("Changed directory to %s", EatErrorReturnString(os.Getwd()))

		}
	}
	cmd.Env = os.Environ()
	VerbosePrintf("Changed directory to %s (After env xfer)", EatErrorReturnString(os.Getwd()))

	var stdoutBuf, stderrBuf bytes.Buffer
	if c.captureStdout {
		cmd.Stdout = &stdoutBuf
	}
	if c.captureStderr {
		cmd.Stderr = &stderrBuf
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.WithStatusCode(-1).WithResponseBody(err.Error())
		return err
	}

	if c.requestPayload != "" {
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, c.requestPayload)
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	if c.showSpinny {
		go c.showSpinner()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Wait for command completion or timeout
	select {
	case <-ctx.Done():
		c.WithStatusCode(-1)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			// Capture output after command completion, even if there's an error
			output := ""
			if c.captureStdout {
				output += stdoutBuf.String()
			}
			if c.captureStderr {
				output += stderrBuf.String()
			}
			c.WithStatusCode(-1).WithResponseBody(output)
			return err
		}
	}

	// Capture output after command has completed successfully
	output := ""
	if c.captureStdout {
		output += stdoutBuf.String()
	}
	if c.captureStderr {
		output += stderrBuf.String()
	}

	c.WithStatusCode(cmd.ProcessState.ExitCode()).WithResponseBody(output)
	//fmt.Println("output: ", c.GetResponseBody())
	//fmt.Println("len of output is: ", len(c.GetResponseBody()))
	//VerbosePrintf("wd is %s", EatErrorReturnString(os.Getwd()))
	//VerbosePrintf("reverting back to  %s", cwd)

	//return os.Chdir(cwd)
	return nil
}

func (c *CLIExecutor) showSpinner() {
	for {
		for _, r := range `-\|/` {
			fmt.Printf("\r%c", r)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func DDHumanReadableStorageSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%dK", size/1024)
	}
	return fmt.Sprintf("%dM", size/1024/1024)
}

func ReduceToBlockSize(size int64, blockSize int64) int64 {
	return (size / blockSize) * blockSize
}

func DiskDuplicatorArgs(device string, outputFile string, blockSize int64, count int64) []string {
	if blockSize == 0 {
		blockSize = 512
	}
	return strings.Split(fmt.Sprintf("if=/dev/%s of=%s bs=%s seek=%d count=1", device, outputFile, DDHumanReadableStorageSize(blockSize), count/blockSize-1), " ")
}
