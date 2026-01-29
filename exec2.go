package alfredo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/shlex"
)

type CLIExecutor struct {
	command           string
	subCommand        string
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
	ctx               context.Context
	secureMode        bool //opt out
	useSudo           bool
}

const DefaultExeTimeout = 5 * time.Second

func NewCLIExecutor() *CLIExecutor {
	ex := &CLIExecutor{}
	//from now on, all instances will be long running by default
	return ex.AsLongRunning()
}

func (c *CLIExecutor) WithCommand(command string) *CLIExecutor {
	c.command = command
	return c
}

func (c *CLIExecutor) WithSubCommand(subcommand string) *CLIExecutor {
	c.subCommand = subcommand
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

func (c *CLIExecutor) HasSSH() bool {
	return len(c.sshHost) != 0
}

func (c *CLIExecutor) DropSSH() *CLIExecutor {
	c.sshHost = ""
	c.sshKey = ""
	c.sshUser = ""
	return c
}

func (c *CLIExecutor) WithSSHStruct(s SSHStruct) *CLIExecutor {
	c.sshHost = s.Host
	c.sshKey = s.Key
	c.sshUser = s.User
	//	c.insecureMode = s.Insecure
	return c
}

func (c *CLIExecutor) WithContext(ctx context.Context) *CLIExecutor {
	c.ctx = ctx
	return c
}

func (c *CLIExecutor) GetSSH() SSHStruct {
	return SSHStruct{
		Host: c.sshHost,
		Key:  c.sshKey,
		User: c.sshUser,
		//		Insecure: c.insecureMode,
	}
}

func (c *CLIExecutor) WithSSHDebug(b bool) *CLIExecutor {
	c.debugSSH = b
	return c
}

func (c *CLIExecutor) InsecureMode() *CLIExecutor {
	c.secureMode = false
	return c
}
func (c *CLIExecutor) SecureMode() *CLIExecutor {
	c.secureMode = true
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

func (c *CLIExecutor) WithSudo(s bool) *CLIExecutor {
	c.useSudo = s
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

func (c *CLIExecutor) GetResponseBodyAsSlice() []string {
	body := c.GetResponseBody()
	if len(body) == 0 {
		return []string{}
	}
	return strings.Split(body, "\n")
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
		return fmt.Sprintf("/usr/bin/ssh -i %s %s@%s %q", ExpandTilde(c.sshKey), c.sshUser, c.sshHost, c.GetCommandWithSub())
	}

	return c.GetCommandWithSub()
}

func (c *CLIExecutor) GetCommandWithSub() string {
	cmd := c.command
	if len(c.subCommand) > 0 && strings.Contains(c.command, "[SUB]") {
		//		fmt.Println("Replacing [SUB] in command")
		cmd = strings.ReplaceAll(c.command, "[SUB]", "\""+c.subCommand+"\"")
	}
	if c.useSudo {
		cmd = "sudo " + cmd
	}
	return cmd
}

func (c *CLIExecutor) GetCommand() string {
	if c.useSudo {
		return "sudo " + c.command
	}
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
			// For SSH, we need to properly escape the directory path
			escapedDir := strings.ReplaceAll(c.directory, "'", "'\"'\"'")
			newcommand := fmt.Sprintf("cd '%s' && %s", escapedDir, c.command)
			c.command = newcommand
		}
	}
	VerbosePrintf("cwd: %s\n", cwd)

	if len(c.sshUser) == 0 {
		c.sshUser = os.Getenv("USER")
	}

	if c.ctx == nil {
		c.ctx = context.Background()
	}

	if c.sshHost != "" {
		arglist := []string{"-i", ExpandTilde(c.sshKey), fmt.Sprintf("%s@%s", c.sshUser, c.sshHost), c.GetCommandWithSub()}
		if c.debugSSH || GetDebug() {
			fmt.Println("key=" + c.sshKey)
			fmt.Println("user=" + c.sshUser)
			fmt.Println("host=" + c.sshHost)
			fmt.Println("command=" + c.command)

			arglist = append([]string{"-vvv"}, arglist...)
		}
		if !c.secureMode {
			VerbosePrintln("using insecure ssh method (no host key checking)")
			arglist = append([]string{"-o", "StrictHostKeyChecking=no",
				"-o", "GlobalKnownHostsFile=/dev/null",
				"-o", "LogLevel=ERROR",
				"-o", "ControlMaster=no",
				"-o", "ControlPath=none",
				"-o", "ControlPersist=no",
				"-o", "UserKnownHostsFile=/dev/null"}, arglist...)
		}

		cmd = exec.CommandContext(c.ctx, "/usr/bin/ssh", arglist...)

		cmdtest := append([]string{"/usr/bin/ssh"}, arglist...)
		VerbosePrintf("exec2.go:: full ssh command: %s\n", strings.Join(cmdtest, " "))

	} else {
		VerbosePrintf("exec2.go:: command: %s\n", c.command)
		VerbosePrintf("exec2.go:: (sub)command: %s\n", c.subCommand)

		// Use shlex to properly parse shell-like arguments
		args, err := shlex.Split(c.GetCommandWithSub())
		if err != nil {
			return fmt.Errorf("failed to parse command arguments: %w", err)
		}

		if len(args) == 0 {
			return fmt.Errorf("no command provided after parsing")
		}
		if GetDebug() {
			VerbosePrintf("exec2.go:: (2) command: %s\n", c.command)
		}
		if len(args) > 1 {
			cmd = exec.CommandContext(c.ctx, args[0], args[1:]...)
		} else {
			cmd = exec.CommandContext(c.ctx, args[0])
		}

		if len(c.directory) > 0 {
			cmd.Dir = c.directory
		}
	}
	cmd.Env = os.Environ()
	if GetDebug() {
		VerbosePrintf("exec2.go:: (3) command: %s\n", c.command)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	if c.captureStdout && !c.dump {
		cmd.Stdout = &stdoutBuf
	} else if c.dump && !c.captureStdout {
		cmd.Stdout = os.Stdout
	} else if c.dump && c.captureStdout {
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	}

	if GetDebug() {
		VerbosePrintf("exec2.go:: (4) command: %s\n", c.command)
	}

	if c.captureStderr && !c.dump {
		cmd.Stderr = &stderrBuf
	} else if c.dump && !c.captureStderr {
		cmd.Stderr = os.Stderr
	} else if c.dump && c.captureStderr {
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	if GetDebug() {
		VerbosePrintf("exec2.go:: (5) command: %s\n", c.command)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.WithStatusCode(-1).WithResponseBody(err.Error())
		VerbosePrintf("exec2.go:: (err=%s) command: %s\n", err.Error(), c.command)

		return err
	}
	if GetDebug() {

		VerbosePrintf("exec2.go:: (6) command: %s\n", c.command)
	}
	if c.requestPayload != "" {
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, c.requestPayload)
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if GetDebug() {
		VerbosePrintf("exec2.go:: (7) command: %s\n", c.command)
	}
	done := make(chan error)
	go func() {

		if GetDebug() {
			VerbosePrintf("exec2.go:: (8) command: %s\n", c.command)
		}
		done <- cmd.Run()
	}()

	if c.showSpinny {
		go c.showSpinner()
	}

	if GetDebug() {
		VerbosePrintf("exec2.go:: (9) command: %s\n", c.command)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {

		if GetDebug() {
			VerbosePrintf("exec2.go:: (10) command: %s\n", c.command)
		}
		<-sigChan
		cancel()
	}()

	// Wait for command completion or timeout
	select {
	case <-ctx.Done():
		if c.ignoreExitCodeOne {
			if cmd.ProcessState.ExitCode() == 1 {
				c.WithStatusCode(0)

				if GetDebug() {
					VerbosePrintf("exec2.go:: (normal termination; ignoring exit code 1) command: %s\n", c.command)
				}

				return nil
			}
		}

		c.WithStatusCode(-1)
		if GetDebug() {
			VerbosePrintf("exec2.go:: (err(2)=%s) command: %s\n", ctx.Err().Error(), c.command)
		}
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

			if GetDebug() {
				if err == nil {
					VerbosePrintf("exec2.go:: (normal termination, ignoring exit code 1) command: %s\n", c.command)
					return nil
				}
				VerbosePrintf("exec2.go:: (err(3)=%s) command: %s\n", err.Error(), c.command)
			}
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
	if GetDebug() {
		VerbosePrintf("exec2.go:: (normal termination) command: %s\n", c.command)
	}
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
	c.WithCommand("md5sum " + fileName).WithCaptureStdout(true).WithCaptureStderr(true).AsLongRunning()

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

func (c *CLIExecutor) NormalizeName(fuzzy string) error {
	if len(fuzzy) == 0 {
		c.responseBody = ""
		return nil
	}
	if !strings.Contains(fuzzy, "*") {
		c.responseBody = fuzzy
		return nil
	}

	if filepath.IsAbs(fuzzy) {
		c.WithDirectory(filepath.Dir(fuzzy))
		fuzzy = filepath.Base(fuzzy)
	}

	// If running over SSH, use 'find' to get matching files and their mtimes, then pick the newest.
	// Otherwise, walk the local filesystem and do the same.
	if len(c.sshHost) > 0 {
		// Use find to list files matching the pattern and their mtimes, sort by mtime descending, pick the first.

		findCmd := fmt.Sprintf(`find . -maxdepth 1 -type f -name %q -printf "%%T@ %%p\n" | sort -nr | head -n1 | awk '{print $2}'`, fuzzy)

		c.WithCommand(findCmd).WithCaptureStdout(true).WithCaptureStderr(true).AsLongRunning()

		//fmt.Println("cli: ", c.GetCli())

		if err := c.Execute(); err != nil {
			return err
		}
	} else {

		// Local version: walk the directory, match pattern, pick newest
		dir := c.directory
		if dir == "" {
			dir = "."
		}
		matches, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		var newestFile string
		var newestModTime time.Time
		for _, entry := range matches {
			if entry.IsDir() {
				continue
			}
			matched, _ := filepath.Match(fuzzy, entry.Name())
			if matched {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if info.ModTime().After(newestModTime) {
					newestModTime = info.ModTime()
					newestFile = entry.Name()
				}
			}
		}
		c.responseBody = newestFile

	}

	result := strings.TrimSpace(c.GetResponseBody())
	result = strings.TrimPrefix(result, "./")
	if len(c.directory) > 0 && len(strings.TrimSpace(result)) != 0 {
		result = filepath.Join(c.directory, result)
	}
	c.responseBody = result
	return nil
}

func (c *CLIExecutor) CreateDirectory(dir string, chown string) error {
	c.WithCommand(fmt.Sprintf("mkdir -p %s", dir)).AsLongRunning()
	if err := c.Execute(); err != nil {
		return fmt.Errorf("unable to create directory %s: %s", dir, err.Error())
	}
	c.WithCommand(fmt.Sprintf("chmod 0755 %s", dir))
	if err := c.Execute(); err != nil {
		return fmt.Errorf("unable to set permissions on directory %s: %s", dir, err.Error())
	}
	if len(chown) > 0 {
		c.WithCommand(fmt.Sprintf("chown %s %s", chown, dir))
		if err := c.Execute(); err != nil {
			return fmt.Errorf("unable to set ownership on directory %s: %s", dir, err.Error())
		}
	}
	return nil
}

func (c *CLIExecutor) TotalSpaceUsed(filter string) error {
	c.AsLongRunning().WithCaptureStdout(true).WithCommand("df -B1 --output=used,target")
	if err := c.Execute(); err != nil {
		return fmt.Errorf("unable to get total space used: %s", err.Error())
	}

	// Parse the output and extract the total space used
	output := strings.TrimSpace(c.GetResponseBody())
	lines := strings.Split(output, "\n")
	totalSpaceUsed := int64(0)

	VerbosePrintf("Found %d lines in df output", len(lines))

	for _, line := range lines {
		VerbosePrintf("Processing line: %s", line)
		if strings.Contains("Used Mounted on", line) {
			VerbosePrintf("Skipping header line: %s", line)
			continue
		}
		if strings.Contains(line, filter) || len(filter) == 0 {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				used, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return fmt.Errorf("unable to parse used space: %s", err.Error())
				}
				totalSpaceUsed += used
				VerbosePrintf("Total space used (filtered by '%s'): %d bytes", filter, totalSpaceUsed)
			} else {
				VerbosePrintf("Skipping line, not enough parts: %s", line)
			}
		} else {
			VerbosePrintf("Skipping line, does not match filter '%s': %s", filter, line)
		}
	}
	c.WithResponseBody(strconv.FormatInt(totalSpaceUsed, 10))
	return nil
}
func (c *CLIExecutor) TotalSpaceAvailable(filter string) error {
	c.AsLongRunning().WithCaptureStdout(true).WithCommand("df -B1 --output=avail,target")
	if err := c.Execute(); err != nil {
		return fmt.Errorf("unable to get total space available: %s", err.Error())
	}

	// Parse the output and extract the total space available
	output := strings.TrimSpace(c.GetResponseBody())
	lines := strings.Split(output, "\n")
	totalSpaceAvailable := int64(0)

	VerbosePrintf("Found %d lines in df output", len(lines))

	for _, line := range lines {
		VerbosePrintf("Processing line: %s", line)
		if strings.Contains("Avail Mounted on", line) {
			VerbosePrintf("Skipping header line: %s", line)
			continue
		}
		if strings.Contains(line, filter) || len(filter) == 0 {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				avail, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return fmt.Errorf("unable to parse available space: %s", err.Error())
				}
				totalSpaceAvailable += avail
				VerbosePrintf("Total space available (filtered by '%s'): %d bytes", filter, totalSpaceAvailable)
			} else {
				VerbosePrintf("Skipping line, not enough parts: %s", line)
			}
		} else {
			VerbosePrintf("Skipping line, does not match filter '%s': %s", filter, line)
		}
	}
	c.WithResponseBody(strconv.FormatInt(totalSpaceAvailable, 10))
	return nil
}

// if err := exe.CountDrivesOverThreshold(threshold, filter, &count); err != nil {
func (c *CLIExecutor) CountDrivesOverThreshold(threshold int, filter string, count *int) error {
	if threshold < 1 || threshold > 99 {
		return fmt.Errorf("threshold must be between 1 and 99")
	}
	c.AsLongRunning().WithCaptureStdout(true).WithCommand("df -B1 --output=pcent,target")
	if err := c.Execute(); err != nil {
		return fmt.Errorf("unable to get drive usage: %s", err.Error())
	}

	// Parse the output and count the drives over the threshold
	output := strings.TrimSpace(c.GetResponseBody())
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, filter) || len(filter) == 0 {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				percent, err := strconv.Atoi(parts[0][:len(parts[0])-1]) // Remove the '%' sign
				if err != nil {
					return fmt.Errorf("unable to parse drive usage percent: %s", err.Error())
				}
				if percent > threshold {
					(*count)++
				}
			}
		}
	}
	return nil
}

func (c *CLIExecutor) GetResponseBodyAsInt64() int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(c.GetResponseBody()), 10, 64)
	return value
}

func (c *CLIExecutor) SSHKeyGen(keyFile string, bits int, passphrase string) *CLIExecutor {
	if len(keyFile) == 0 {
		panic("keyFile cannot be empty")
	}
	if bits < 1024 {
		bits = 4096
	}
	return c.WithCommand("ssh-keygen -t rsa -b " + strconv.Itoa(bits) + " -f " + keyFile + " -N " + passphrase + " -q").AsLongRunning()
}

// func installPublicKey(user, host, publicKey string) error {
// 	// Command to append key to authorized_keys on remote
// 	cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", user, host),
// 		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '"+publicKey+"' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")

// 	var stderr bytes.Buffer
// 	cmd.Stderr = &stderr

// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to install public key: %v, stderr: %s", err, stderr.String())
// 	}
// 	return nil
// }

func (c *CLIExecutor) InstallPublicKeyFileToRemote(publicKeyFile string) error {
	publicKey := string(EatErrorReturnBytes(os.ReadFile(publicKeyFile)))
	return c.InstallPublicKeyToRemote(publicKey)
}

func (c *CLIExecutor) InstallPublicKeyToRemote(publicKey string) error {
	c.WithCommand(fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", publicKey)).AsLongRunning()
	return c.Execute()
}

func (c *CLIExecutor) Gzip(filePath string) error {
	c.WithCommand("gzip " + filePath).AsLongRunning()
	return c.Execute()
}

func (c *CLIExecutor) Gunzip(filePath string) error {
	c.WithCommand("gunzip " + filePath).AsLongRunning()
	return c.Execute()
}

func (exe *CLIExecutor) TailLog(logPath string) *CLIExecutor {
	exe.WithCommand("tail -10 " + logPath).
		AsLongRunning().
		WithCaptureStdout(true).
		WithSpinny(true)
	return exe
}

// TailLogAndWaitForKeyword tails the last 10 lines of a log file over SSH
// until it finds the keyword or returns an error if execution fails.
func (exe *CLIExecutor) WaitForKeyword(keyword string, interval time.Duration) error {

	exe.AsLongRunning().WithCaptureStdout(true).WithSpinny(true)

	for {
		if err := exe.Execute(); err != nil {
			return err
		}

		if strings.Contains(exe.GetResponseBody(), keyword) {
			return nil
		}

		time.Sleep(interval)
	}
}

func (exe *CLIExecutor) WaitForKeywordInLog(logPath, keyword string, interval time.Duration) error {
	return exe.TailLog(logPath).WithTimeout(10*time.Second).WaitForKeyword(keyword, interval)
}

func (exe *CLIExecutor) BackgroundedCommand(command string, outputFile string) *CLIExecutor {
	if len(outputFile) == 0 {
		outputFile = "/dev/null"
	}
	//like WithCommand, but adds nohup and & to run in background and leave it there
	return exe.WithCommand(fmt.Sprintf("nohup %s > %s 2>&1 </dev/null & echo $!", command, outputFile))
}

func (exe *CLIExecutor) ResponseHasKeyword(keyword string) bool {
	return strings.Contains(exe.GetResponseBody(), keyword)
}

func (exe *CLIExecutor) ProcAlive(pid int) bool {
	exe.AsLongRunning().WithCaptureStdout(true).WithSpinny(true)
	exe.WithCommand(fmt.Sprintf("ps -o stat= -p %d", pid)) // Fetch only the process state
	if err := exe.Execute(); err != nil {
		return false
	}

	state := strings.TrimSpace(exe.GetResponseBody())
	if state == "" || strings.HasPrefix(state, "Z") { // Ignore zombie processes
		return false
	}
	return true
}

func (exe *CLIExecutor) WatchForProcessToDie(pid int, checkInterval time.Duration) error {
	exe.AsLongRunning().WithCaptureStdout(true).WithSpinny(true)

	for {
		if !exe.ProcAlive(pid) {
			VerbosePrintf("Process %d has exited.", pid)
			return nil
		}
		VerbosePrintf("Process %d is still running...", pid)

		time.Sleep(checkInterval)
	}
}

func (exe *CLIExecutor) CheckAndKillProcess(procName string) error {
	if len(procName) == 0 {
		return fmt.Errorf("process name is empty")
	}

	exe.WithDirectory("/tmp").
		WithSSHDebug(false).
		WithCaptureStdout(true).
		WithSpinny(false).AsLongRunning().WithCommand("ps -eo pid,command")
	if err := exe.Execute(); err != nil {
		fmt.Println("failed to get process list")
		return err
	}

	//step 2: parse the process list and look for the process name

	slice := strings.Split(exe.GetResponseBody(), "\n")
	for i := 0; i < len(slice); i++ {
		if strings.Contains(slice[i], procName) {
			//step 3: if we find the process, kill it
			//step 4: get the pid
			//step 5: kill the process
			VerbosePrintf("line: %q", slice[i])
			fmt.Println("found process ", procName)
			splits := strings.Split(strings.TrimSpace(slice[i]), " ")
			pid := splits[0]
			fmt.Printf("\tkilling process %q pid= %q\n", procName, pid)
			exe.WithDirectory("/tmp").
				WithSSHDebug(false).
				WithCaptureStdout(true).
				WithSpinny(false).AsLongRunning().WithCommand("kill -9 " + pid)
			if err := exe.Execute(); err != nil {
				fmt.Println("\tfailed to kill process ", procName, " pid=", pid)
				return err
			}
			fmt.Printf("\tkilled process %s with pid=%s on host %s\n", procName, pid, exe.GetSSH().Host)
			return nil
		}
	}
	fmt.Println("did not find process ", procName)
	return nil
}
