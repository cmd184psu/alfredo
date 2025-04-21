package alfredo

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"time"
)

type ExecCallBackFunc func(*string, string) error
type ProgressCallBackFunc func(chan bool)
type WatcherCallBackFunc func(*ExecStruct) error

type CaptureType int

const (
	CapNone CaptureType = iota
	CapBoth
	CapStdout
	CapStderr
)

type ExecStruct struct {
	mainExecFunc     ExecCallBackFunc
	mainCli          string
	watcherExecFunc  WatcherCallBackFunc
	watcherPayload   []byte
	progressExecFunc ProgressCallBackFunc
	spinny           bool
	capture          CaptureType
	iface            interface{}
	dir              string
	SpinSigChan      chan bool
	WatchSigChan     chan bool
	ErrChan          chan error
	body             string
	ssh              SSHStruct
	useSSH           bool
	request          string
}

func (ex *ExecStruct) Init() *ExecStruct {
	ex.dir = "."
	ex.spinny = false
	ex.watcherExecFunc = nil
	ex.mainCli = ""
	ex.mainExecFunc = System3toCapturedString
	ex.SpinSigChan = nil
	ex.WatchSigChan = nil
	ex.ErrChan = nil
	ex.progressExecFunc = Spinny
	ex.capture = CapNone
	ex.body = ""
	ex.useSSH = false

	return ex
}

func (ex *ExecStruct) WithMainExecFunc(cb ExecCallBackFunc, cli string) *ExecStruct {
	ex.mainExecFunc = cb
	ex.mainCli = cli
	return ex
}
func (ex *ExecStruct) WithWatcherExecFunc(cb WatcherCallBackFunc, payload []byte) *ExecStruct {
	ex.watcherExecFunc = cb
	ex.watcherPayload = payload
	return ex
}

func (ex ExecStruct) GetWatcherPayload() []byte {
	return ex.watcherPayload
}

func (ex *ExecStruct) WithProgressExecFunc(cb ProgressCallBackFunc) *ExecStruct {
	ex.progressExecFunc = cb
	return ex
}

func (ex *ExecStruct) WithSpinny(b bool) *ExecStruct {
	ex.spinny = b
	return ex
}

func (ex *ExecStruct) WithHintInterface(i interface{}) *ExecStruct {
	ex.iface = i
	return ex
}
func (ex *ExecStruct) WithDirectory(d string) *ExecStruct {
	ex.dir = d
	return ex
}
func (ex *ExecStruct) WithCapture(c bool) *ExecStruct {
	if c {
		ex.capture = CapBoth
	} else {
		ex.capture = CapNone
	}
	return ex
}

func (ex *ExecStruct) WithCaptureBoth() *ExecStruct {
	ex.capture = CapBoth
	return ex
}

func (ex *ExecStruct) WithRequest(r string) *ExecStruct {
	ex.request = r
	return ex
}

func (ex ExecStruct) GetRequest() string {
	return ex.request
}

func (ex *ExecStruct) WithSSH(ssh SSHStruct) *ExecStruct {
	ex.ssh = ssh
	ex.mainExecFunc = nil
	ex.useSSH = true
	return ex
}

func (ex ExecStruct) GetSSH() SSHStruct {
	return ex.ssh
}

func (ex ExecStruct) GetMainCli() string {
	return ex.mainCli
}

func (ex *ExecStruct) WithMainCli(cli string) *ExecStruct {
	ex.mainCli = cli
	return ex
}
func (ex ExecStruct) GetIface() interface{} {
	return ex.iface
}

func (ex ExecStruct) OkToSpin() bool {
	return ex.spinny && ex.progressExecFunc != nil
}

func (ex ExecStruct) OkToWatch() bool {
	return ex.watcherExecFunc != nil
}

func (ex ExecStruct) GetBody() string {
	return ex.body
}
func (ex ExecStruct) captureOutput(cmd *exec.Cmd) (string, string, error) {
	var stdoutBuf, stderrBuf strings.Builder
	if ex.capture == CapBoth || ex.capture == CapStdout {
		cmd.Stdout = &stdoutBuf
	}
	if ex.capture == CapBoth || ex.capture == CapStderr {
		cmd.Stderr = &stderrBuf
	}
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

/*
	func pipeCommandExecution(requestPayload string, commandToExecute string) (string, error) {
		// Split the command and arguments
		cmdParts := strings.Fields(commandToExecute)
		if len(cmdParts) == 0 {
			return "", fmt.Errorf("empty command")
		}

		// Create a pipe
		pipeReader, pipeWriter := io.Pipe()

		// Create the command
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Stdin = pipeReader

		// Create a buffer to capture the output
		var outputBuffer bytes.Buffer
		cmd.Stdout = &outputBuffer

		// Start the command
		err := cmd.Start()
		if err != nil {
			return "", fmt.Errorf("error starting command: %w", err)
		}

		// Write the payload to the pipe in a separate goroutine
		go func() {
			defer pipeWriter.Close()
			io.WriteString(pipeWriter, requestPayload)
		}()

		// Wait for the command to finish
		err = cmd.Wait()
		if err != nil {
			return "", fmt.Errorf("error running command: %w", err)
		}

		// Return the output as a string
		return outputBuffer.String(), nil
	}
*/

func (ex *ExecStruct) Execute() error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", ex.mainCli)
		return nil
	}
	VerbosePrintln("Execute:: begin")
	var err error
	var wg sync.WaitGroup

	if ex.mainExecFunc == nil && !ex.useSSH {
		panic("missing exec call back function; ssh not configured")
	}
	if ex.spinny && ex.progressExecFunc != nil {
		ex.SpinSigChan = make(chan bool)
	}
	if ex.OkToWatch() {
		ex.WatchSigChan = make(chan bool)
	}
	ex.ErrChan = make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ex.ErrChan)
		if ex.OkToSpin() {
			defer close(ex.SpinSigChan)
		}
		if ex.OkToWatch() {
			defer close(ex.WatchSigChan)
		}
		VerbosePrintln("execute:: inside closure, about to run command")
		var e error
		if ex.useSSH {
			ex.ssh = ex.ssh.WithRemoteDir(ex.dir)
			VerbosePrintln(ex.ssh.GetSSHCli() + " \"" + ex.mainCli + "\"")
			if len(ex.request) > 0 {
				ex.ssh.SetRequest(ex.request)
			}
			e = ex.ssh.Execute(ex.mainCli)
			ex.body = ex.ssh.stdout
		} else {
			cmd := exec.Command("sh", "-c", ex.mainCli)
			if len(ex.request) > 0 {
				pipeReader, pipeWriter := io.Pipe()
				cmd.Stdin = pipeReader
				go func() {
					defer pipeWriter.Close()
					io.WriteString(pipeWriter, ex.request)
				}()
			}
			stdout, stderr, err := ex.captureOutput(cmd)
			if ex.capture == CapBoth || ex.capture == CapStdout {
				ex.body = stdout
			}
			if ex.capture == CapBoth || ex.capture == CapStderr {
				ex.body += stderr
			}
			e = err
			VerbosePrintln("mainExecFunc completed")
		}
		if ex.capture == CapNone && e == nil {
			_, e = io.Copy(os.Stdout, strings.NewReader(ex.body))
		}
		VerbosePrintln("execute:: inside closure, cmd is complete")
		if ex.spinny {
			ex.SpinSigChan <- true
		}
		ex.ErrChan <- e
	}()
	if ex.OkToSpin() {
		go ex.progressExecFunc(ex.SpinSigChan)
	}
	if ex.OkToWatch() {
		go ex.watcherExecFunc(ex)
	}
	err = <-ex.ErrChan
	wg.Wait()

	VerbosePrintln("execute:: the wait is over")
	return err
}

func (ex *ExecStruct) ExecuteSTILLBROKEN() error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", ex.mainCli)
		return nil
	}
	VerbosePrintln("Execute:: begin")
	var err error
	var wg sync.WaitGroup

	if ex.mainExecFunc == nil && !ex.useSSH {
		panic("missing exec call back function; ssh not configured")
	}
	if ex.spinny && ex.progressExecFunc != nil {
		ex.SpinSigChan = make(chan bool)
	}
	if ex.OkToWatch() {
		ex.WatchSigChan = make(chan bool)
	}
	ex.ErrChan = make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ex.ErrChan)
		if ex.OkToSpin() {
			defer close(ex.SpinSigChan)
		}
		if ex.OkToWatch() {
			defer close(ex.WatchSigChan)
		}
		VerbosePrintln("execute:: inside closure, about to run command")
		var e error
		if ex.useSSH {
			ex.ssh = ex.ssh.WithRemoteDir(ex.dir)
			VerbosePrintln(ex.ssh.GetSSHCli() + " \"" + ex.mainCli + "\"")
			e = ex.ssh.SecureRemoteExecution(ex.mainCli)
			ex.body = ex.ssh.stdout
		} else {
			cmd := exec.Command("sh", "-c", ex.mainCli)
			if len(ex.request) > 0 {
				pipeReader, pipeWriter := io.Pipe()
				cmd.Stdin = pipeReader
				go func() {
					defer pipeWriter.Close()
					io.WriteString(pipeWriter, ex.request)
				}()
			}
			stdout, stderr, err := ex.captureOutput(cmd)
			if ex.capture == CapBoth || ex.capture == CapStdout {
				ex.body = stdout
			}
			if ex.capture == CapBoth || ex.capture == CapStderr {
				ex.body += stderr
			}
			e = err
			VerbosePrintln("mainExecFunc completed")
			time.Sleep(5 * time.Second)
		}
		if ex.capture == CapNone && e == nil {
			_, e = io.Copy(os.Stdout, strings.NewReader(ex.body))
		}
		VerbosePrintln("execute:: inside closure, cmd is complete")
		if ex.spinny {
			ex.SpinSigChan <- true
		}
		ex.ErrChan <- e
	}()
	if ex.OkToSpin() {
		go ex.progressExecFunc(ex.SpinSigChan)
	}
	if ex.OkToWatch() {
		go ex.watcherExecFunc(ex)
	}
	err = <-ex.ErrChan
	wg.Wait()

	VerbosePrintln("execute:: the wait is over")
	return err
}
func (ex *ExecStruct) ExecutePRE17Jan2025() error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", ex.mainCli)
		return nil
	}

	VerbosePrintln("Execute:: begin")
	var err error
	var wg sync.WaitGroup

	if ex.mainExecFunc == nil && !ex.useSSH {
		panic("missing exec call back function; ssh not configured")
	}
	if ex.spinny && ex.progressExecFunc != nil {
		ex.SpinSigChan = make(chan bool)
	}
	if ex.OkToWatch() {
		ex.WatchSigChan = make(chan bool)
	}
	ex.ErrChan = make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ex.ErrChan)
		if ex.OkToSpin() {
			defer close(ex.SpinSigChan)
		}
		if ex.OkToWatch() {
			defer close(ex.WatchSigChan)
		}
		VerbosePrintln("execute:: inside closure, about to run command")
		var e error
		var pipeReader *io.PipeReader
		var pipeWriter *io.PipeWriter
		if ex.useSSH {
			ex.ssh = ex.ssh.WithRemoteDir(ex.dir)
			VerbosePrintln(ex.ssh.GetSSHCli() + " \"" + ex.mainCli + "\"")

			e = ex.ssh.SecureRemoteExecution(ex.mainCli)
			ex.body = ex.ssh.stdout
		} else {

			if len(ex.request) > 0 {
				pipeReader, pipeWriter = io.Pipe()

			}
			ex.WithDirectory(ex.dir)
			cmd := exec.Command("sh", "-c", ex.mainCli)

			if len(ex.request) > 0 {
				cmd.Stdin = pipeReader
				go func() {
					defer pipeWriter.Close()
					io.WriteString(pipeWriter, ex.request)
				}()
			}

			stdout, stderr, err := ex.captureOutput(cmd)
			if ex.capture == CapBoth || ex.capture == CapStdout {
				ex.body = stdout
			}
			if ex.capture == CapBoth || ex.capture == CapStderr {
				ex.body += stderr
			}
			e = err
			VerbosePrintln("mainExecFunc completed")
			time.Sleep(5 * time.Second)
		}
		if ex.capture == CapNone && e == nil {
			_, e = io.Copy(os.Stdout, strings.NewReader(ex.body))
		}
		VerbosePrintln("execute:: inside closure, cmd is complete")
		if ex.spinny {
			ex.SpinSigChan <- true
		}
		ex.ErrChan <- e
	}()
	if ex.OkToSpin() {
		go ex.progressExecFunc(ex.SpinSigChan)
	}
	if ex.OkToWatch() {
		go ex.watcherExecFunc(ex)
	}
	err = <-ex.ErrChan
	wg.Wait()

	VerbosePrintln("execute:: the wait is over")
	return err
}
func (ex *ExecStruct) ExecuteOLD() error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", ex.mainCli)
		return nil
	}
	VerbosePrintln("Execute:: begin")
	var err error
	var wg sync.WaitGroup

	if ex.mainExecFunc == nil && !ex.useSSH {
		panic("missing exec call back function; ssh not configured")
	}

	if ex.spinny && ex.progressExecFunc != nil {
		ex.SpinSigChan = make(chan bool)
	}
	if ex.OkToWatch() {
		ex.WatchSigChan = make(chan bool)
	}
	ex.ErrChan = make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ex.ErrChan)
		if ex.OkToSpin() {
			defer close(ex.SpinSigChan)
		}
		if ex.OkToWatch() {
			defer close(ex.WatchSigChan)
		}
		VerbosePrintln("execute:: inside closure, about to run command")
		var e error
		if ex.useSSH {
			ex.ssh = ex.ssh.WithRemoteDir(ex.dir)
			VerbosePrintln(ex.ssh.GetSSHCli() + " \"" + ex.mainCli + "\"")

			e = ex.ssh.SecureRemoteExecution(ex.mainCli)
			ex.body = ex.ssh.stdout
		} else {
			ex.WithDirectory(ex.dir)
			e = ex.mainExecFunc(&ex.body, ex.mainCli)
			VerbosePrintln("mainExecFunc completed")
			time.Sleep(5 * time.Second)
		}
		// if !ex.capture && e == nil {
		// 	_, e = io.Copy(os.Stdout, strings.NewReader(ex.body))
		// }
		VerbosePrintln("execute:: inside closure, cmd is complete")
		if ex.spinny {
			ex.SpinSigChan <- true
		}
		// if ex.OkToWatch() {
		// 	ex.WatchSigChan <- true
		// }
		ex.ErrChan <- e
	}()
	if ex.OkToSpin() {
		go ex.progressExecFunc(ex.SpinSigChan)
	}
	//wg.Add(1)
	if ex.OkToWatch() {
		go ex.watcherExecFunc(ex)
	}
	err = <-ex.ErrChan
	wg.Wait()

	VerbosePrintln("execute:: the wait is over")
	return err
}

func LocalExecuteAndSpin(cli string) error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cli)
		return nil
	}
	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)

		e := System3(cli)
		sigChan <- true
		errorChan <- e
	}()
	go Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return err
}

func RunToLess(cmd1 *exec.Cmd) error {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd1)
		return nil
	}
	//cmd1 := exec.Command("myprogram", "-help")

	// Create the command for `less`
	cmd2 := exec.Command("less")

	// Create a pipe for the output of `myprogram -help`
	cmd1Out, err := cmd1.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating stdout pipe for cmd1: %v", err)
	}

	// Set the stdin of `less` to the stdout of `myprogram -help`
	cmd2.Stdin = cmd1Out

	// Start the first command
	if err := cmd1.Start(); err != nil {
		return fmt.Errorf("Error starting cmd1: %v", err)
	}

	// Start the second command
	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("Error starting cmd2: %v", err)
	}

	// Wait for the first command to finish
	if err := cmd1.Wait(); err != nil {
		return fmt.Errorf("Error waiting for cmd1: %v", err)
	}

	// Wait for the second command to finish
	if err := cmd2.Wait(); err != nil {
		return fmt.Errorf("Error waiting for cmd2: %v", err)
	}
	return nil
}

func GoFuncAndSpin(cb interface{}, params ...interface{}) error {
	if cb == nil {
		return fmt.Errorf("callback function is nil")
	}

	cbValue := reflect.ValueOf(cb)
	if cbValue.Kind() != reflect.Func {
		return fmt.Errorf("callback is not a function")
	}

	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)

		// Use reflection to call the callback function with the provided parameters
		cbValue := reflect.ValueOf(cb)
		cbParams := make([]reflect.Value, len(params))
		for i, param := range params {
			cbParams[i] = reflect.ValueOf(param)
		}
		results := cbValue.Call(cbParams)

		// Assume the callback function returns a single error
		if len(results) > 0 && results[0].CanInterface() {
			if e, ok := results[0].Interface().(error); ok {
				err = e
			}
		}

		sigChan <- true
		errorChan <- err
	}()
	if !GetQuiet() {
		go Spinny(sigChan)
	}
	err = <-errorChan
	wg.Wait()
	return err
}
