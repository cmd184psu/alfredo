package alfredo

import (
	"fmt"
	"os"
	"testing"
	"time"
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

	if err = ssh.RemoteExecuteAndSpin(cli); err != nil {
		t.Errorf("remote execute and spin failed with %s", err.Error())
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

	if err = ssh.RemoteExecuteAndSpin("stat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("---")
	fmt.Println(ssh.GetBody())
	fmt.Println("---")
	if err = ssh.RemoteExecuteAndSpin("cat " + remotefile); err != nil {
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

	if err = ssh.RemoteExecuteAndSpin("stat " + remotefile); err != nil {
		t.Errorf(err.Error())
	}
	fmt.Println("---")
	fmt.Println(ssh.GetBody())
	fmt.Println("---")
	if err = ssh.RemoteExecuteAndSpin("cat " + remotefile); err != nil {
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
				RemoteDir: tt.fields.remoteDir,
				silent:    tt.fields.silent,
				exitCode:  tt.fields.exitCode,
			}
			if err := s.WriteSparseFile(tt.args.f, tt.args.sizeMin, tt.args.sizeMax, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.WriteSparseFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// func TestSSHStruct_CrossCopy(t *testing.T) {
// 	var sshtgt SSHStruct
// 	sshtgt.Key = ExpandTilde("~/.ssh/homelab_rsa")
// 	sshtgt.Host = "192.168.1.33"
// 	sshtgt.User = "root"

// 	type fields struct {
// 		Key       string
// 		User      string
// 		Host      string
// 		capture   bool
// 		stdout    string
// 		stderr    string
// 		port      int
// 		RemoteDir string
// 		silent    bool
// 		exitCode  int
// 	}
// 	type args struct {
// 		srcFile string
// 		tgtssh  SSHStruct
// 		tgtFile string
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "base test",
// 			fields: fields{
// 				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
// 				User: "cdelezenski",
// 				Host: "builder.cmdhome.net",
// 			},
// 			args: args{
// 				srcFile: "",
// 				tgtssh:  sshtgt,
// 				tgtFile: "",
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			srcssh := &SSHStruct{
// 				Key:       tt.fields.Key,
// 				User:      tt.fields.User,
// 				Host:      tt.fields.Host,
// 				capture:   tt.fields.capture,
// 				stdout:    tt.fields.stdout,
// 				stderr:    tt.fields.stderr,
// 				port:      tt.fields.port,
// 				RemoteDir: tt.fields.RemoteDir,
// 				silent:    tt.fields.silent,
// 				exitCode:  tt.fields.exitCode,
// 			}
// 			if err := srcssh.CrossCopy(tt.args.srcFile, tt.args.tgtssh, tt.args.tgtFile); (err != nil) != tt.wantErr {
// 				t.Errorf("SSHStruct.CrossCopy() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

func TestSSHStruct_CrossCopy(t *testing.T) {
	var sshtgt SSHStruct
	sshtgt.Key = ExpandTilde("~/.ssh/homelab_rsa")
	sshtgt.Host = "192.168.1.33"
	sshtgt.User = "root"

	type fields struct {
		Key       string
		User      string
		Host      string
		capture   bool
		stdout    string
		stderr    string
		port      int
		RemoteDir string
		silent    bool
		exitCode  int
		ccmode    CrossCopyModeType
	}
	type args struct {
		srcFile string
		tgtssh  SSHStruct
		tgtFile string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "base test - temp file",
			fields: fields{
				Key:    ExpandTilde("~/.ssh/homelab_rsa"),
				User:   "cdelezenski",
				Host:   "builder.cmdhome.net",
				ccmode: GetCCTypeOf("temp"),
			},
			args: args{
				srcFile: "",
				tgtssh:  sshtgt,
				tgtFile: "",
			},
		},
		{
			name: "base test - via shell",
			fields: fields{
				Key:    ExpandTilde("~/.ssh/homelab_rsa"),
				User:   "cdelezenski",
				Host:   "builder.cmdhome.net",
				ccmode: CCMVIASHELL,
			},
			args: args{
				srcFile: "",
				tgtssh:  sshtgt,
				tgtFile: "",
			},
		},
		{
			name: "base test - via memory",
			fields: fields{
				Key:    ExpandTilde("~/.ssh/homelab_rsa"),
				User:   "cdelezenski",
				Host:   "builder.cmdhome.net",
				ccmode: CCMVIAMEMORY,
			},
			args: args{
				srcFile: "",
				tgtssh:  sshtgt,
				tgtFile: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcssh := &SSHStruct{
				Key:       tt.fields.Key,
				User:      tt.fields.User,
				Host:      tt.fields.Host,
				capture:   tt.fields.capture,
				stdout:    tt.fields.stdout,
				stderr:    tt.fields.stderr,
				port:      tt.fields.port,
				RemoteDir: tt.fields.RemoteDir,
				silent:    tt.fields.silent,
				exitCode:  tt.fields.exitCode,
				ccmode:    tt.fields.ccmode,
			}
			if err := srcssh.CrossCopy(tt.args.srcFile, tt.args.tgtssh, tt.args.tgtFile); (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.CrossCopyInMemory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSHStruct_SyncFileWithRemote(t *testing.T) {
	//generate some files; files should contain the same content as their name
	tmp := os.Getenv("HOME") + "/tmp"
	WriteLineToFile(tmp+"/remotefile-older.txt", "remotefile-older")
	time.Sleep(1 * time.Second)
	WriteLineToFile(tmp+"/localfile-older.txt", "localfile-older")
	time.Sleep(1 * time.Second)
	if err := WriteLineToFile(tmp+"/localfile.txt", "localfile"); err != nil {
		t.Errorf("failed to write localfile.txt: %s", err.Error())
	}
	time.Sleep(1 * time.Second)
	WriteLineToFile(tmp+"/remotefile.txt", "remotefile")
	//WriteLineToFile("/tmp/remotefile-dne.txt", "remotefile-dne")
	time.Sleep(1 * time.Second)
	WriteLineToFile(tmp+"/remotefile-newer.txt", "remotefile-newer")
	time.Sleep(1 * time.Second)

	type fields struct {
		Key            string
		User           string
		Host           string
		capture        bool
		stdout         string
		stderr         string
		port           int
		RemoteDir      string
		silent         bool
		exitCode       int
		ccmode         CrossCopyModeType
		ConnectTimeout int
		request        string
	}
	type args struct {
		localFile         string
		remoteFile        string
		hashValidation    bool
		createDirectories bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "sync valid local file to remote",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/localfile.txt",
				remoteFile:        tmp + "/remotefile.txt",
				hashValidation:    true,
				createDirectories: true,
			},
			wantErr: false,
		},
		{
			name: "sync non-existent local file to remote",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/nonexistentfile.txt",
				remoteFile:        tmp + "/remotefile.txt",
				hashValidation:    true,
				createDirectories: false,
			},
			wantErr: false,
		},
		{

			//test fails, but it should not.  The file is there, but ssh fails horribly trying to md5sum it
			name: "sync valid local file to non-existent remote",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/localfile.txt",
				remoteFile:        tmp + "/jumbolia/remotefile-dne.txt",
				hashValidation:    true,
				createDirectories: true,
			},
			wantErr: false,
		},
		{
			name: "sync valid local file (newer) to remote (older)",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/localfile.txt",
				remoteFile:        tmp + "/remotefile-older.txt",
				hashValidation:    false,
				createDirectories: false,
			},
			wantErr: false,
		},

		{
			name: "sync valid local file (older) to remote (newer)",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/localfile-older.txt",
				remoteFile:        tmp + "/remotefile-newer.txt",
				hashValidation:    false,
				createDirectories: false,
			},
			wantErr: false,
		},
		{
			name: "sync valid local file to invalid remote path",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				Host: "localhost",
				port: 22,
			},
			args: args{
				localFile:         tmp + "/localfile.txt",
				remoteFile:        "/invalidpath/remotefile.txt",
				hashValidation:    true,
				createDirectories: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SSHStruct{
				Key:            tt.fields.Key,
				User:           tt.fields.User,
				Host:           tt.fields.Host,
				capture:        tt.fields.capture,
				stdout:         tt.fields.stdout,
				stderr:         tt.fields.stderr,
				port:           tt.fields.port,
				RemoteDir:      tt.fields.RemoteDir,
				silent:         tt.fields.silent,
				exitCode:       tt.fields.exitCode,
				ccmode:         tt.fields.ccmode,
				ConnectTimeout: tt.fields.ConnectTimeout,
				request:        tt.fields.request,
			}

			fmt.Printf("syncronize %s with %s over ssh\n", tt.args.localFile, tt.args.remoteFile)
			fmt.Printf("\tcat %s\n", tt.args.localFile)
			fmt.Printf("\tcat %s\n", tt.args.remoteFile)
			if err := s.SyncFileWithRemote(tt.args.localFile, tt.args.remoteFile, tt.args.hashValidation, tt.args.createDirectories); (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.SyncFileWithRemote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSHStruct_RemoteFileCount(t *testing.T) {
	type fields struct {
		Key            string
		User           string
		Host           string
		capture        bool
		stdout         string
		stderr         string
		port           int
		RemoteDir      string
		silent         bool
		exitCode       int
		ccmode         CrossCopyModeType
		ConnectTimeout int
		request        string
	}
	type args struct {
		sdirectoryPath string
		prefix         string
		glob           string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "count 3 files in output",
			fields: fields{
				Key:  ExpandTilde("~/.ssh/homelab_rsa"),
				User: os.Getenv("USER"),
				Host: "localhost",
			},
			args: args{
				sdirectoryPath: "/tmp/testing",
				prefix:         "",
				glob:           "*.txt",
			},
			want:    3, // Split will return 4: 3 files + 1 empty string after last newline
			wantErr: false,
		},
		// {
		// 	name: "count 1 file in output",
		// 	fields: fields{
		// 		stdout: "file1.txt\n",
		// 	},
		// 	args: args{
		// 		sdirectoryPath: "/tmp",
		// 		prefix:         "",
		// 		glob:           "*.txt",
		// 	},
		// 	want:    2, // 1 file + 1 empty string
		// 	wantErr: false,
		// },
		// {
		// 	name: "empty output",
		// 	fields: fields{
		// 		stdout: "",
		// 	},
		// 	args: args{
		// 		sdirectoryPath: "/tmp",
		// 		prefix:         "",
		// 		glob:           "*.txt",
		// 	},
		// 	want:    1, // Split("") returns [""]
		// 	wantErr: false,
		// },
		// {
		// 	name: "output with no trailing newline",
		// 	fields: fields{
		// 		stdout: "file1.txt\nfile2.txt\nfile3.txt",
		// 	},
		// 	args: args{
		// 		sdirectoryPath: "/tmp",
		// 		prefix:         "",
		// 		glob:           "*.txt",
		// 	},
		// 	want:    3,
		// 	wantErr: false,
		// },
		// {
		// 	name: "output with extra blank lines",
		// 	fields: fields{
		// 		stdout: "file1.txt\n\nfile2.txt\n\nfile3.txt\n",
		// 	},
		// 	args: args{
		// 		sdirectoryPath: "/tmp",
		// 		prefix:         "",
		// 		glob:           "*.txt",
		// 	},
		// 	want:    6, // 3 files + 3 empty lines
		// 	wantErr: false,
		// },
	}
	exe := NewCLIExecutor()
	if err := exe.WithCommand("mkdir /tmp/testing").Execute(); err != nil {
		t.Errorf("failed to create /tmp/testing: %s", err.Error())
	}
	for i := 0; i < 3; i++ {
		if err := exe.WithCommand(fmt.Sprintf("touch /tmp/testing/file%d.txt", i)).Execute(); err != nil {
			t.Errorf("failed to create /tmp/testing/file%d.txt: %s", i, err.Error())
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssh := SSHStruct{
				Key:            tt.fields.Key,
				User:           tt.fields.User,
				Host:           tt.fields.Host,
				capture:        tt.fields.capture,
				stdout:         tt.fields.stdout,
				stderr:         tt.fields.stderr,
				port:           tt.fields.port,
				RemoteDir:      tt.fields.RemoteDir,
				silent:         tt.fields.silent,
				exitCode:       tt.fields.exitCode,
				ccmode:         tt.fields.ccmode,
				ConnectTimeout: tt.fields.ConnectTimeout,
				request:        tt.fields.request,
			}
			got, err := ssh.RemoteFileCount(tt.args.sdirectoryPath, tt.args.prefix, tt.args.glob)
			fmt.Println("got=", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.RemoteFileCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SSHStruct.RemoteFileCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHStruct_GetRemoteFileSize(t *testing.T) {
	tmpDir := os.TempDir()
	testFile := tmpDir + "/testfile_size.txt"
	testContent := []byte("this is a test file for size check\nwith multiple lines\n")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Get local file size for comparison
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}
	localSize := stat.Size()

	// Prepare SSHStruct for localhost
	ssh := &SSHStruct{
		Key:  ExpandTilde("~/.ssh/homelab_rsa"),
		User: os.Getenv("USER"),
		Host: "localhost",
		port: 22,
	}

	// Simulate remote command output by setting stdout to the file size as string
	// In a real scenario, you would run a remote command like "stat -c %s <file>"
	// Here, we simulate as if the remote command has been run and output captured
	ssh.stdout = fmt.Sprintf("%d", localSize)

	tests := []struct {
		name       string
		remoteFile string
		want       int64
		wantErr    bool
	}{
		{
			name:       "existing file, correct size",
			remoteFile: testFile,
			want:       localSize,
			wantErr:    false,
		},
		{
			name:       "non-existent file",
			remoteFile: tmpDir + "/does_not_exist.txt",
			want:       0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "existing file, correct size":
				ssh.stdout = fmt.Sprintf("%d", localSize)
			case "non-existent file":
				ssh.stdout = "no such file"
			}
			got, err := ssh.GetRemoteFileSize(tt.remoteFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSHStruct.GetRemoteFileSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("SSHStruct.GetRemoteFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
