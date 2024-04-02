package alfredo

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type ExecCallBackFunc func(*string, string) error
type ProgressCallBackFunc func(chan bool)
type WatcherCallBackFunc func(*ExecStruct) error

type ExecStruct struct {
	mainExecFunc     ExecCallBackFunc
	mainCli          string
	watcherExecFunc  WatcherCallBackFunc
	progressExecFunc ProgressCallBackFunc
	spinny           bool
	capture          bool
	iface            interface{}
	dir              string
	SpinSigChan      chan bool
	WatchSigChan     chan bool
	ErrChan          chan error
	body             string
	ssh              SSHStruct
	useSSH           bool
}

func (ex ExecStruct) Init() ExecStruct {
	ex.dir = "."
	ex.spinny = false
	ex.watcherExecFunc = nil
	ex.mainCli = ""
	ex.mainExecFunc = System3toCapturedString
	ex.SpinSigChan = nil
	ex.WatchSigChan = nil
	ex.ErrChan = nil
	ex.progressExecFunc = Spinny
	ex.capture = false
	ex.body = ""
	ex.useSSH = false

	return ex
}

func (ex ExecStruct) WithMainExecFunc(cb ExecCallBackFunc, cli string) ExecStruct {
	ex.mainExecFunc = cb
	ex.mainCli = cli
	return ex
}
func (ex ExecStruct) WithWatcherExecFunc(cb WatcherCallBackFunc) ExecStruct {
	ex.watcherExecFunc = cb
	return ex
}
func (ex ExecStruct) WithProgressExecFunc(cb ProgressCallBackFunc) ExecStruct {
	ex.progressExecFunc = cb
	return ex
}

func (ex ExecStruct) WithSpinny(b bool) ExecStruct {
	ex.spinny = b
	return ex
}

func (ex ExecStruct) WithHintInterface(i interface{}) ExecStruct {
	ex.iface = i
	return ex
}
func (ex ExecStruct) WithDirectory(d string) ExecStruct {
	ex.dir = d
	return ex
}
func (ex ExecStruct) WithCapture(c bool) ExecStruct {
	ex.capture = c
	return ex
}

func (ex ExecStruct) WithSSH(ssh SSHStruct, cli string) ExecStruct {
	ex.ssh = ssh
	ex.mainCli = cli
	ex.mainExecFunc = nil
	ex.useSSH = true
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
func (ex *ExecStruct) Execute() error {
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
			e = ex.mainExecFunc(&ex.body, ex.mainCli)
			VerbosePrintln("mainExecFunc completed")
			time.Sleep(5 * time.Second)
		}
		if !ex.capture && e == nil {
			_, e = io.Copy(os.Stdout, strings.NewReader(ex.body))
		}
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
