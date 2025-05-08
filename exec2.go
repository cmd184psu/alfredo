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
	command           string
	requestPayload    string
	sshHost           string
	sshKey            string
	sshUser           string
	timeout           time.Duration
	showSpinny        bool
	captureStdout     bool
	captureStderr     bool
	statusCode        int
	responseBody      string
	trimWhiteSpace    bool
	directory         string
	dump              bool
	debugSSH          bool
	ignoreExitCodeOne bool
}

const default_timeout = 5 * time.Second

func NewCLIExecutor() *CLIExecutor {
	ex := &CLIExecutor{}
	t := default_timeout
	//ex.SetTimeout(t)
	return ex.WithTimeout(t)
}

func (c *CLIExecutor) WithCommand(command string) *CLIExecutor {
	c.command = command
	return c
}

func (c *CLIExecutor) WithRequestPayload(payload string) *CLIExecutor {
	c.requestPayload = payload
	return c
}

func (c *CLIExecutor) WithSSH(host, key, user string) *CLIExecutor {
	c.sshHost = host
	c.sshKey = key
	c.sshUser = user
	return c
}

func (c *CLIExecutor) WithSSHStruct(s SSHStruct) *CLIExecutor {
	c.sshHost = s.Host
	c.sshKey = s.Key
	c.sshUser = s.User
	return c
}

func (c *CLIExecutor) GetSSH() SSHStruct {
	return SSHStruct{
		Host: c.sshHost,
		Key:  c.sshKey,
		User: c.sshUser,
	}
}

func (c *CLIExecutor) WithSSHDebug(b bool) *CLIExecutor {
	c.debugSSH = b
	return c
}

func (c *CLIExecutor) WithDirectory(directory string) *CLIExecutor {
	c.directory = directory
	return c
}
func (c *CLIExecutor) WithExitOneIsOK() *CLIExecutor {
	c.ignoreExitCodeOne = true
	return c
}
func (c *CLIExecutor) WithExitOneIsNotOK() *CLIExecutor {
	c.ignoreExitCodeOne = false
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

func (c *CLIExecutor) DumpOutput() *CLIExecutor {
	c.captureStdout = true
	c.captureStderr = true
	c.dump = true
	return c
}

func (c *CLIExecutor) WithSpinny(show bool) *CLIExecutor {
	if GetQuiet() {
		c.showSpinny = false
		return c
	}
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

func (c *CLIExecutor) GetRequestPayload() string {
	return c.requestPayload
}

func (c *CLIExecutor) GetCli() string {
	if c.sshHost != "" {
		if len(c.sshUser) == 0 {
			c.sshUser = os.Getenv("USER")
		}
		return fmt.Sprintf("/usr/bin/ssh -i %s %s@%s %q", ExpandTilde(c.sshKey), c.sshUser, c.sshHost, c.command)
	}
	return c.command
}

func (c *CLIExecutor) GetCommand() string {
	return c.command
}

func (c *CLIExecutor) GetDirectory() string {
	return c.directory
}
func (c *CLIExecutor) CreateSymlink(fromFile, toLink string) error {

	if len(c.sshHost) > 0 {
		c.AsLongRunning().WithCommand(fmt.Sprintf("ln -s %s %s", fromFile, toLink))
		return c.DumpOutput().Execute()
	}

	cwd := EatErrorReturnString(os.Getwd())

	if len(c.directory) > 0 {
		if err := os.Chdir(c.directory); err != nil {
			return fmt.Errorf("chdir to %s failed: %s", c.directory, err.Error())
		}
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				panic(fmt.Sprintf("chdir to %s failed: %s", cwd, err.Error()))
			}
		}()
	}
	err := os.Symlink(fromFile, toLink)
	if err != nil {
		return fmt.Errorf("unable to create symlink %s -> %s: %s", fromFile, toLink, err.Error())
	}

	return nil
}

func (c *CLIExecutor) Execute() error {
	if c.debugSSH || GetDebug() {
		SetVerbose(true)
	}
	if GetDryRun() {
		fmt.Println("DRYRUN: ", c.GetCli())
		return nil
	}
	var cmd *exec.Cmd

	cwd := EatErrorReturnString(os.Getwd())

	if len(cwd) == 0 {
		return fmt.Errorf("current working directory is empty")
	}

	defer func() {
		if err := os.Chdir(cwd); err != nil {
			panic(fmt.Sprintf("chdir to %s failed: %s", cwd, err.Error()))
		}
	}()

	//only chdir if local
	if len(c.sshHost) == 0 && len(c.directory) > 0 {
		if err := os.Chdir(c.directory); err != nil {
			return fmt.Errorf("chdir to %q failed: %s", c.directory, err.Error())
		}
	}
	if len(c.command) == 0 {
		return fmt.Errorf("no command provided")
	}
	if len(c.directory) > 0 {
		if len(c.sshHost) > 0 {
			newcommand := "cd " + c.directory + " && " + c.command
			c.command = newcommand
		}
	}
	VerbosePrintf("cwd: %s\n", cwd)
	// if len(c.directory) > 0 {
	// 	VerbosePrintf("want to chdir to %s\n", c.directory)
	// } else {
	// 	VerbosePrintf("don't change directories\n")
	// }

	// VerbosePrintf("ssh host= %s\n", c.sshHost)
	// VerbosePrintf("ssh user= %s\n", c.sshUser)
	// VerbosePrintf("ssh key= %s\n", c.sshKey)
	// VerbosePrintf("command: %s\n", c.command)
	// VerbosePrintf("args: %s\n", strings.Join(c.args, " "))
	if len(c.sshUser) == 0 {
		c.sshUser = os.Getenv("USER")
	}

	if c.sshHost != "" {
		if c.debugSSH || GetDebug() {
			fmt.Println("key=" + c.sshKey)
			fmt.Println("user=" + c.sshUser)
			fmt.Println("host=" + c.sshHost)
			fmt.Println("commmand=" + c.command)
			cmd = exec.Command("/usr/bin/ssh", "-vvv", "-i", ExpandTilde(c.sshKey), fmt.Sprintf("%s@%s", c.sshUser, c.sshHost), c.command)
		} else {
			cmd = exec.Command("/usr/bin/ssh", "-i", ExpandTilde(c.sshKey), fmt.Sprintf("%s@%s", c.sshUser, c.sshHost), c.command)
		}
	} else {
		VerbosePrintf("command: %s\n", c.command)
		split := strings.Split(c.command, " ")
		if len(split) > 1 {
			cmd = exec.Command(split[0], split[1:]...)
		} else {
			cmd = exec.Command(c.command)
		}
		if len(c.directory) > 0 {
			cmd.Dir = c.directory
		}
	}
	cmd.Env = os.Environ()
	//VerbosePrintf("Changed directory to %s (After env xfer)", EatErrorReturnString(os.Getwd()))

	var stdoutBuf, stderrBuf bytes.Buffer
	if c.captureStdout && !c.dump {
		cmd.Stdout = &stdoutBuf
	} else if c.dump && !c.captureStdout {
		cmd.Stdout = os.Stdout
	} else if c.dump && c.captureStdout {
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	}
	if c.captureStderr && !c.dump {
		cmd.Stderr = &stderrBuf
	} else if c.dump && !c.captureStderr {
		cmd.Stderr = os.Stderr
	} else if c.dump && c.captureStderr {
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
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
		if c.ignoreExitCodeOne {
			if cmd.ProcessState.ExitCode() == 1 {
				c.WithStatusCode(0)
				return nil
			}
		}

		c.WithStatusCode(-1)
		return ctx.Err()
	case err := <-done:
		if err != nil {

			if c.ignoreExitCodeOne && cmd.ProcessState.ExitCode() == 1 {
				c.WithStatusCode(0)
				err = nil
			}

			// Capture output after command completion, even if there's an error
			output := ""
			if c.captureStdout {
				output += stdoutBuf.String()
			}
			if c.captureStderr {
				output += stderrBuf.String()
			}
			c.WithStatusCode(-1).WithResponseBody(output)
			// fmt.Println("err:", err)
			// fmt.Println("output:", output)
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

func (c *CLIExecutor) HashFile(fileName string) string {
	if len(fileName) == 0 {
		return ""
	}
	//does not make sense!

	c.WithCommand("md5sum " + fileName).WithCaptureStdout(true).WithCaptureStderr(true)

	if err := c.Execute(); err != nil {
		panic("Error: " + err.Error())
	}

	split := strings.Split(c.GetResponseBody(), " ")
	if len(split) < 2 {
		return ""
	}
	return split[0]
}

// use this instead of execute or local java processes
func (c *CLIExecutor) CaptureJavaProcessList(jvm string) error {
	VerbosePrintf("BEGIN:: CaptureJavaProcessList(%s)", jvm)
	defer VerbosePrintf("END:: CaptureJavaProcessList(%s)", jvm)
	c.WithCaptureStdout(true).WithCaptureStderr(true)
	c.WithCommand("ps -eo pid,command").WithResponseBody("").WithTrimWhiteSpace(true).AsLongRunning()

	//fmt.Println("cli: ", c.GetCli())

	if err := c.Execute(); err != nil {
		return err
	}

	if len(c.responseBody) == 0 {
		return fmt.Errorf("CaptureJavaProcessList():: no response body")
	}

	//VerbosePrintf("response body: %s", c.responseBody)

	proclist, err := GetJavaProcessesFromBytes([]byte(c.responseBody), jvm)

	if err != nil {
		return err
	}

	c.responseBody = PrettyPrint(proclist)
	return nil
}

func (c *CLIExecutor) GetProcListFromResponseBody() []ProcessInfo {
	var proclist []ProcessInfo
	if err := ReadStructFromString(c.responseBody, &proclist); err != nil {
		return []ProcessInfo{}
	}
	return proclist
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
	seek := count/blockSize - 1
	if seek < 0 {
		seek = 0
	}
	return strings.Split(fmt.Sprintf("if=/dev/%s of=%s bs=%s seek=%d count=1", device, outputFile, DDHumanReadableStorageSize(blockSize), seek), " ")
}

// nc -zv 192.168.1.100 80 && echo "Port is open" || echo "Port is closed"
func (c *CLIExecutor) IsThisPortOpenIPV4(ip string, port int) *CLIExecutor {
	return c.WithCommand(fmt.Sprintf("nc -zv %s %d", ip, port)).WithExitOneIsOK().WithCaptureStdout(true).WithCaptureStderr(true).AsLongRunning()
}

func (c *CLIExecutor) IsPortOpen() bool {
	if !c.ignoreExitCodeOne {
		panic("coding mistake, should call IsThisPortOpenIPV4() first")
	}
	c.WithExitOneIsNotOK()
	if c.statusCode == 0 && strings.Contains(c.GetResponseBody(), "Connected to") {
		return true
	}
	return false
}
