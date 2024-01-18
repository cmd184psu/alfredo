package alfredo

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExecutiveNull(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Executive, improperly initialized, did not panic")
		}
	}()
	var exe ExecStruct
	exe.Execute()
}
func TestExecutiveCapturedNothing(t *testing.T) {
	var exe ExecStruct
	exe = exe.Init().
		WithMainExecFunc(System3toCapturedString, "/usr/bin/true").
		WithSpinny(false).
		WithCapture(true).
		WithDirectory(".")
	exe.Execute()
	if len(exe.body) > 0 {
		t.Errorf("expected no body")
	}
}
func TestExecutiveCapturedEverything(t *testing.T) {
	var exe ExecStruct
	exe = exe.Init().
		WithMainExecFunc(System3toCapturedString, "find ./").
		WithSpinny(false).
		WithCapture(true).
		WithDirectory(".")
	exe.Execute()
	if len(exe.body) == 0 {
		t.Errorf("expected content")
	}
}

func TestExecutiveNotExecutable(t *testing.T) {
	var exe ExecStruct
	exe = exe.Init().
		WithMainExecFunc(System3toCapturedString, "./exec.go").
		WithSpinny(false).
		WithCapture(true).
		WithDirectory(".")
	e := exe.Execute()
	if e == nil {
		t.Errorf("Attempted to execute non-executable file, successfully -- shouldn't happen!")
	} else if !strings.Contains(e.Error(), "permission denied") {
		t.Errorf("threw wrong error for this condition: %s", e.Error())

	}
}

func TestExecutiveOverSSH(t *testing.T) {
	var ssh SSHStruct

	ssh.SetDefaults()

	ReadStructFromJSONFile("./ssh.json", &ssh)
	if ssh.Key[0] == '~' {
		ssh.Key = os.Getenv("HOME") + ssh.Key[1:]
	}
	fmt.Println(PrettyPrint(ssh))

	var exe ExecStruct
	exe = exe.Init().
		WithSSH(ssh, "ls -lah").
		WithSpinny(true).
		WithCapture(false).
		WithDirectory(".")
	e := exe.Execute()
	if e != nil {
		t.Errorf("failed to execute over ssh with error: %s", e.Error())
	}
}

type testCaseStruct struct {
	input ExecStruct
	note  string
	err   error
}

type dumbStruct struct {
	item1 bool
	cli   string
}

func donothing(ex *ExecStruct) error {
	quit := false
	n := ex.GetIface().(dumbStruct)
	for !quit {
		time.Sleep(1 * time.Second)
		fmt.Println("donothing says: " + TrueIsYes(n.item1))
		e := System3(n.cli)
		if e != nil {
			panic(e.Error())
		}
		select {
		case sig := <-ex.WatchSigChan:
			quit = sig //fmt.Println("received signal", sig)
		default:
			//fmt.Println("not yet")
		}

	}
	VerbosePrintln("leaving donothing")
	return nil
}

func TestExecutiveSpinNoWatch(t *testing.T) {
	var ex ExecStruct
	var d dumbStruct
	var tc testCaseStruct
	var testCases []testCaseStruct
	cli := "stat exec.go"
	d.cli = "ls -lah"
	tc.input = ex.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithWatcherExecFunc(donothing).
		WithHintInterface(d).
		WithSpinny(false).
		WithDirectory(".")
	tc.note = "no spin, with watcher"
	tc.err = nil

	testCases = append(testCases, tc)

	tc.input = ex.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithWatcherExecFunc(donothing).
		WithHintInterface(d).
		WithSpinny(true).
		WithDirectory(".")
	tc.note = "with spin, with watcher"
	tc.err = nil

	testCases = append(testCases, tc)

	tc.input = ex.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithSpinny(true).
		WithDirectory(".")
	tc.note = "with spin, no watcher"
	tc.err = nil

	testCases = append(testCases, tc)

	tc.input = ex.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithSpinny(false).
		WithDirectory(".")
	tc.note = "no spin, no watcher"
	tc.err = nil

	testCases = append(testCases, tc)

	tc.input = ex.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithSpinny(false).
		WithCapture(true).
		WithDirectory(".")
	tc.note = "no spin, no watcher, capture on"
	tc.err = nil

	testCases = append(testCases, tc)

	for _, tc := range testCases {

		fmt.Println("\n=======================================")
		fmt.Println("\tTesting " + tc.note)
		fmt.Println("\n=======================================")
		RemoveFile("./complete")
		RemoveFile("./killingit")
		result := tc.input.Execute()
		if tc.input.capture && len(tc.input.body) == 0 {
			t.Errorf("collection was empty (fail) for %s", tc.note)
		}

		if result != tc.err {
			t.Errorf("execute test failed on %s", tc.note)
		}
	}
}

func TestMain(m *testing.M) {
	SetVerbose(true)
	// Run the tests
	os.Exit(m.Run())
}
