package alfredo

import (
	"fmt"
	"os"
	"testing"
)

// func TestExecutiveOverSSH(t *testing.T) {
// 	var ssh SSHStruct

// 	ssh.SetDefaults()
// 	ReadStructFromJSONFile("./ssh.json", &ssh)

// 	fmt.Println(PrettyPrint(ssh))

// 	var exe ExecStruct
// 	exe = exe.Init().
// 		WithSSH(ssh, "ls -lah").
// 		WithSpinny(true).
// 		WithCapture(false).
// 		WithDirectory(".")
// 	e := exe.Execute()
// 	if e != nil {
// 		t.Errorf("failed to execute over ssh with error: %s", e.Error())
// 	}
// }

// func TestAdminApi(t *testing.T) {
// 	var as AdminApiStruct

// 	as.IPAddress = "192.168.1.41"

// 	//	as.Password = "dXGS+hQEXKeLsMBahKVNodN9txg="
// 	as.GroupId = "spullg3"
// 	as.UserId = "spullu3"
// 	as.sshEnabled = true
// 	as.ssh.Host = "192.168.1.41"
// 	as.ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
// 	as.ssh.User = "root"

// 	if e := as.AcquirePasswordRemotely(as.ssh); e != nil {
// 		t.Errorf("failed to acquire password over ssh")
// 	}

// 	if !as.ssh.RemoteFileExists("/etc/hosts") {
// 		t.Errorf("non test related ssh failure")
// 	}

// 	gs, e := as.GetGroupList()
// 	if e != nil {
// 		t.Errorf("create user api call failed over ssh")
// 	}
// 	if len(gs) == 0 {
// 		t.Errorf("group list was empty")

// 	}
// }

// func TestAdminApi2(t *testing.T) {
// 	var as AdminApiStruct

// 	as.IPAddress = "192.168.1.41"

// 	//	as.Password = "dXGS+hQEXKeLsMBahKVNodN9txg="
// 	as.GroupId = "spullg3"
// 	as.UserId = "spullu3"
// 	as.sshEnabled = true
// 	as.ssh.Host = "192.168.1.41"
// 	as.ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
// 	as.ssh.User = "root"

// 	if e := as.AcquirePasswordRemotely(as.ssh); e != nil {
// 		t.Errorf("failed to acquire password over ssh")
// 	}

// 	if !as.GroupExists() {
// 		t.Errorf("failed to find group " + as.GroupId + " over ssh")
// 	}

// }

// func TestAdminApi3(t *testing.T) {

// 	var as AdminApiStruct

// 	as.IPAddress = "192.168.1.41"

// 	//	as.Password = "dXGS+hQEXKeLsMBahKVNodN9txg="
// 	as.GroupId = "spullg3"
// 	as.UserId = "spullu3"
// 	as.sshEnabled = true
// 	as.ssh.Host = "192.168.1.41"
// 	as.ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
// 	as.ssh.User = "root"

// 	if e := as.AcquirePasswordRemotely(as.ssh); e != nil {
// 		t.Errorf("failed to acquire password over ssh")
// 	}

// 	gl, err := as.GetGroupList()

// 	if err != nil {
// 		t.Errorf("failed: " + err.Error())
// 	}

// 	if len(gl) == 0 {
// 		t.Errorf("length of group list is zero")
// 	}
// 	for i := 0; i < len(gl); i++ {
// 		fmt.Printf("gl[%d].GroupName=%s\n", i, gl[i].GroupName)
// 		fmt.Printf("gl[%d].GroupId=%s\n", i, gl[i].GroupId)
// 	}
// }

func TestSSHlistOpt(t *testing.T) {
	fmt.Println("---TEST SSH LIST of /opt----")
	var ssh SSHStruct
	var err error
	ssh.Host = "192.168.1.41"
	ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
	ssh.User = "root"
	ssh.SetRemoteDir("/opt")
	cli := "ls -lah"

	if ssh, err = ssh.RemoteExecuteAndSpin(cli); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println(ssh.GetBody())
	fmt.Println("---END TEST SSH LIST of /opt----")
}

func TestSSHPipeContentToSSHCLI(t *testing.T) {
	fmt.Println("---TEST SSH pipe test ----")
	var ssh SSHStruct
	var err error
	ssh.Host = "192.168.1.41"
	ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
	ssh.User = "root"
	remotefile := "/root/remotefile-pipe.txt"
	content := "and a one, and a two and a ..."
	//	ssh.SetRemoteDir("/opt")
	cli := "tee " + remotefile

	if err = ssh.SecureRemotePipeExecution([]byte(content), cli); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println(ssh.GetBody())

	if ssh, err = ssh.RemoteExecuteAndSpin("stat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("---")
	fmt.Println(ssh.GetBody())
	fmt.Println("---")
	if ssh, err = ssh.RemoteExecuteAndSpin("cat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println(ssh.GetBody())

	fmt.Println("---END TEST SSH LIST of /opt----")
}

func TestSSHContentToRemoteFile(t *testing.T) {
	fmt.Println("---TEST CONTENT to remote file test ----")

	var ssh SSHStruct
	var err error
	ssh.Host = "192.168.1.41"
	ssh.Key = os.Getenv("HOME") + "/.ssh/homelab_rsa"
	ssh.User = "root"
	remotefile := "/root/remotefile.txt"
	content := "and a one, and a two and a ..."

	if err := ssh.SecureUploadContent([]byte(content), remotefile); err != nil {
		t.Errorf(err.Error())
	}

	if ssh, err = ssh.RemoteExecuteAndSpin("stat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("---")
	fmt.Println(ssh.GetBody())
	fmt.Println("---")
	if ssh, err = ssh.RemoteExecuteAndSpin("cat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println(ssh.GetBody())
	fmt.Println("---END: TEST CONTENT to remote file test ----")
}

// func TestMain(m *testing.M) {
// 	alfredo.SetVerbose(true)
// 	// Run the tests
// 	os.Exit(m.Run())
// }

func TestSSHStruct_WriteSparseFile(t *testing.T) {
	type fields struct {
		Key       string
		User      string
		Host      string
		capture   bool
		stdout    string
		stderr    string
		port      int
		remoteDir string
		silent    bool
		exitCode  int
	}
	type args struct {
		f       string
		sizeMin int
		sizeMax int
		r       int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "base test",
			fields: fields{},
			args: args{
				f:       "./localfile.dat",
				sizeMin: 2,
				sizeMax: 4,
				r:       2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SSHStruct{
				Key:       tt.fields.Key,
				User:      tt.fields.User,
				Host:      tt.fields.Host,
				capture:   tt.fields.capture,
				stdout:    tt.fields.stdout,
				stderr:    tt.fields.stderr,
				port:      tt.fields.port,
				remoteDir: tt.fields.remoteDir,
				silent:    tt.fields.silent,
				exitCode:  tt.fields.exitCode,
			}
			if err := s.WriteSparseFile(tt.args.f, tt.args.sizeMin, tt.args.sizeMax, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.WriteSparseFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
