package alfredo

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLIExecutor_Execute(t *testing.T) {
	SetQuiet(true)
	tests := []struct {
		name           string
		command        string
		requestPayload string
		sshHost        string
		sshKey         string
		sshUser        string
		directory      string
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
			command:      "sleep 10",
			timeout:      1 * time.Second,
			showSpinny:   true,
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
			name:          "current directory",
			command:       "pwd",
			directory:     "/usr/local/bin",
			captureStdout: true,
			wantOutput:    "/usr/local/bin",
			wantExitCode:  0,
			wantErr:       false,
		},
		{
			name:          "remote execute with directory, single command",
			command:       "pwd",
			directory:     "/usr/local/bin",
			captureStdout: true,
			sshHost:       "localhost",
			sshKey:        ExpandTilde("~/.ssh/homelab_rsa"),
			sshUser:       "root",

			wantOutput:   "/usr/local/bin",
			wantExitCode: 0,
			wantErr:      false,
		},
		{
			name:    "remote execute without directory, single command",
			command: "pwd",
			//			directory:     "/usr/local/bin",
			captureStdout: true,
			sshHost:       "localhost",
			sshKey:        ExpandTilde("~/.ssh/homelab_rsa"),
			sshUser:       "root",

			wantOutput:   "/root",
			wantExitCode: 0,
			wantErr:      false,
		},
		{
			name:           "local md5sum 2",
			command:        "./md5sum",
			directory:      "/usr/bin",
			requestPayload: "5555",
			captureStdout:  true,
			wantOutput:     "6074c6aa3488f3c2dddff2a7ca821aab  -",
			wantExitCode:   0,
			wantErr:        false,
		},
		{
			name:           "remote md5sum 2",
			sshHost:        "localhost",
			sshKey:         ExpandTilde("~/.ssh/homelab_rsa"),
			sshUser:        os.Getenv("USER"),
			command:        "./md5sum",
			directory:      "/usr/bin",
			captureStdout:  true,
			requestPayload: "5555",
			wantOutput:     "6074c6aa3488f3c2dddff2a7ca821aab  -",
			wantExitCode:   0,
			wantErr:        false,
		},
		{
			name:           "remote md5sum bad directory",
			sshHost:        "localhost",
			sshKey:         ExpandTilde("~/.ssh/homelab_rsa"),
			sshUser:        os.Getenv("USER"),
			command:        "./md5sum",
			directory:      "c:/Program Files",
			captureStdout:  true,
			requestPayload: "5555",
			wantOutput:     "",
			wantExitCode:   -1,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewCLIExecutor().
				WithCommand(tt.command).
				WithRequestPayload(tt.requestPayload).
				WithSSH(tt.sshHost, tt.sshKey, tt.sshUser).
				WithSSHStruct(SSHStruct{
					User: tt.sshUser,
					Key:  tt.sshKey,
					Host: tt.sshHost,
				}).
				WithDirectory(tt.directory).
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
			fmt.Println("cli: ", executor.GetCli())
			if gotOutput != tt.wantOutput {
				t.Errorf("CLIExecutor.Execute() gotOutput = %q, want %q", gotOutput, tt.wantOutput)
			}
			if gotExitCode != tt.wantExitCode {
				t.Errorf("CLIExecutor.Execute() gotExitCode = %v, want %v", gotExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestDiskDuplicatorArgs(t *testing.T) {
	type args struct {
		device     string
		outputFile string
		blockSize  int64
		count      int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Small block size",
			args: args{
				device:     "sda",
				outputFile: "output.img",
				blockSize:  512,
				count:      1024,
			},
			want: "if=/dev/sda of=output.img bs=512 seek=1 count=1",
		},
		{
			name: "Medium block size",
			args: args{
				device:     "sda",
				outputFile: "output.img",
				blockSize:  1024 * 1024,
				count:      1024 * 1024 * 10,
			},
			want: "if=/dev/sda of=output.img bs=1M seek=9 count=1",
		},
		{
			name: "Large block size",
			args: args{
				device:     "sda",
				outputFile: "output.img",
				blockSize:  1024 * 1024,
				count:      1024 * 1024 * 1024 * 5,
			},
			want: "if=/dev/sda of=output.img bs=1M seek=5119 count=1",
		},
		{
			name: "Exact block size",
			args: args{
				device:     "sda",
				outputFile: "output.img",
				blockSize:  1024,
				count:      1024,
			},
			want: "if=/dev/sda of=output.img bs=1K seek=0 count=1",
		},
		{
			name: "Non-divisible block size",
			args: args{
				device:     "sda",
				outputFile: "output.img",
				blockSize:  1500,
				count:      3000,
			},
			want: "if=/dev/sda of=output.img bs=1K seek=1 count=1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DiskDuplicatorArgs(tt.args.device, tt.args.outputFile, tt.args.blockSize, tt.args.count); strings.Join(got, " ") != tt.want {
				t.Errorf("DiskDuplicatorArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCLIExecutor_HashFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		sshHost  string
		sshKey   string
		sshUser  string
		wantHash string
		wantErr  bool
	}{
		{
			name:     "Local file hash",
			fileName: ExpandTilde("~/tmp/localfile.txt"),
			wantHash: "b2c6cae0ec5ea5624307f40e92fa22c2", // Replace with the actual hash of the file
			wantErr:  false,
		},
		{
			name:     "Remote file hash",
			fileName: ExpandTilde("~/tmp/localfile.txt"),
			sshHost:  "localhost",
			sshKey:   ExpandTilde("~/.ssh/homelab_rsa"),
			sshUser:  os.Getenv("USER"),
			wantHash: "b2c6cae0ec5ea5624307f40e92fa22c2", // Replace with the actual hash of the file
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewCLIExecutor().
				WithSSH(tt.sshHost, tt.sshKey, tt.sshUser)

			gotHash := executor.HashFile(tt.fileName)
			if (gotHash == "") != tt.wantErr {
				t.Errorf("CLIExecutor.HashFile() error = %v, wantErr %v", gotHash == "", tt.wantErr)
				return
			}
			if gotHash != tt.wantHash {
				t.Errorf("CLIExecutor.HashFile() = %v, want %v", gotHash, tt.wantHash)
			}
		})
	}
}

func TestCLIExecutor_CreateSymlink(t *testing.T) {
	type fields struct {
		command        string
		args           []string
		requestPayload string
		sshHost        string
		sshKey         string
		sshUser        string
		timeout        time.Duration
		showSpinny     bool
		captureStdout  bool
		captureStderr  bool
		statusCode     int
		responseBody   string
		trimWhiteSpace bool
		directory      string
		dump           bool
	}
	type args struct {
		fromFile string
		toLink   string
	}

	ex := NewCLIExecutor()
	ex.WithCommand("touch /tmp/source.txt").Execute()
	ex.WithCommand("rm /tmp/link.txt").Execute()
	ex.WithCommand("rm /tmp/remotelink.txt").Execute()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Local symlink creation",
			fields: fields{
				directory: "/tmp",
			},
			args: args{
				fromFile: "/tmp/source.txt",
				toLink:   "/tmp/link.txt",
			},
			wantErr: false,
		},
		{
			name: "Remote symlink creation",
			fields: fields{
				sshHost: "localhost",
				sshKey:  ExpandTilde("~/.ssh/homelab_rsa"),
				sshUser: os.Getenv("USER"),
			},
			args: args{
				fromFile: "/tmp/source.txt",
				toLink:   "/tmp/remotelink.txt",
			},
			wantErr: false,
		},
		{
			name: "Local symlink creation with non-existent source",
			fields: fields{
				directory: "/tmp",
			},
			args: args{
				fromFile: "/tmp/nonexistent.txt",
				toLink:   "/tmp/link.txt",
			},
			wantErr: true,
		},
		{
			name: "Remote symlink creation with non-existent source",
			fields: fields{
				sshHost: "localhost",
				sshKey:  ExpandTilde("~/.ssh/homelab_rsa"),
				sshUser: os.Getenv("USER"),
			},
			args: args{
				fromFile: "/tmp/nonexistent.txt",
				toLink:   "/tmp/link.txt",
			},
			wantErr: true,
		},
		{
			name: "Local symlink creation with invalid directory",
			fields: fields{
				directory: "/invalid/directory",
			},
			args: args{
				fromFile: "/tmp/source.txt",
				toLink:   "/tmp/link.txt",
			},
			wantErr: true,
		},
		{
			name: "Remote symlink creation with invalid SSH key",
			fields: fields{
				sshHost: "localhost",
				sshKey:  "/invalid/key",
				sshUser: os.Getenv("USER"),
			},
			args: args{
				fromFile: "/tmp/source.txt",
				toLink:   "/tmp/link.txt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CLIExecutor{
				command:        tt.fields.command,
				requestPayload: tt.fields.requestPayload,
				sshHost:        tt.fields.sshHost,
				sshKey:         tt.fields.sshKey,
				sshUser:        tt.fields.sshUser,
				timeout:        tt.fields.timeout,
				showSpinny:     tt.fields.showSpinny,
				captureStdout:  tt.fields.captureStdout,
				captureStderr:  tt.fields.captureStderr,
				statusCode:     tt.fields.statusCode,
				responseBody:   tt.fields.responseBody,
				trimWhiteSpace: tt.fields.trimWhiteSpace,
				directory:      tt.fields.directory,
				dump:           tt.fields.dump,
			}
			if err := c.CreateSymlink(tt.args.fromFile, tt.args.toLink); (err != nil) != tt.wantErr {
				t.Errorf("CLIExecutor.CreateSymlink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCLIExecutor_CaptureJavaProcessList(t *testing.T) {
	var ssh SSHStruct
	if err := ssh.Load("./testbox.json"); err != nil {
		t.Errorf("Failed to load SSH config: %v", err)
	}
	fmt.Println("ssh=", PrettyPrint(ssh))
	// Set the SSH key and user from the loaded config
	SetVerbose(true)
	type fields struct {
		command        string
		requestPayload string
		sshHost        string
		sshKey         string
		sshUser        string
		timeout        time.Duration
		showSpinny     bool
		captureStdout  bool
		captureStderr  bool
		statusCode     int
		responseBody   string
		trimWhiteSpace bool
		directory      string
		dump           bool
		debugSSH       bool
	}
	type args struct {
		jvm string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// {
		// 	name: "Capture Java processes locally",
		// 	fields: fields{
		// 		captureStdout: true,
		// 		captureStderr: true,
		// 		debugSSH:      true,
		// 	},
		// 	args: args{
		// 		jvm: "",
		// 	},
		// 	wantErr: false,
		// },
		{
			name: "Capture all Java processes remotely",
			fields: fields{
				sshHost:       ssh.Host,
				sshKey:        ExpandTilde(ssh.Key),
				sshUser:       ssh.User,
				captureStdout: true,
				captureStderr: true,
			},
			args: args{
				jvm: "",
			},
			wantErr: false,
		},
		// {
		// 	name: "Capture Java processes with invalid SSH key",
		// 	fields: fields{
		// 		sshHost:       "localhost",
		// 		sshKey:        "/invalid/key",
		// 		sshUser:       os.Getenv("USER"),
		// 		captureStdout: true,
		// 		captureStderr: true,
		// 	},
		// 	args: args{
		// 		jvm: "java",
		// 	},
		// 	wantErr: true,
		// },

		{
			name: "Capture Specific Java processes",
			fields: fields{
				sshHost:       ssh.Host,
				sshKey:        ExpandTilde(ssh.Key),
				sshUser:       ssh.User,
				captureStdout: true,
				captureStderr: true,
			},
			args: args{
				jvm: "BucketMigration",
			},
			wantErr: false,
		},
		// {
		// 	name: "Capture Java processes remotely with invalid host",
		// 	fields: fields{
		// 		sshHost:       "invalidhost",
		// 		sshKey:        ExpandTilde("~/.ssh/homelab_rsa"),
		// 		sshUser:       os.Getenv("USER"),
		// 		captureStdout: true,
		// 		captureStderr: true,
		// 	},
		// 	args: args{
		// 		jvm: "java",
		// 	},
		// 	wantErr: true,
		// },
		// {
		// 	name: "Capture Java processes with debug mode enabled",
		// 	fields: fields{
		// 		debugSSH:      true,
		// 		captureStdout: true,
		// 		captureStderr: true,
		// 	},
		// 	args: args{
		// 		jvm: "java",
		// 	},
		// 	wantErr: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CLIExecutor{
				command:        tt.fields.command,
				requestPayload: tt.fields.requestPayload,
				sshHost:        tt.fields.sshHost,
				sshKey:         tt.fields.sshKey,
				sshUser:        tt.fields.sshUser,
				timeout:        tt.fields.timeout,
				showSpinny:     tt.fields.showSpinny,
				captureStdout:  tt.fields.captureStdout,
				captureStderr:  tt.fields.captureStderr,
				statusCode:     tt.fields.statusCode,
				responseBody:   tt.fields.responseBody,
				trimWhiteSpace: tt.fields.trimWhiteSpace,
				directory:      tt.fields.directory,
				dump:           tt.fields.dump,
				debugSSH:       tt.fields.debugSSH,
			}
			if err := c.CaptureJavaProcessList(tt.args.jvm); (err != nil) != tt.wantErr {
				t.Errorf("CLIExecutor.CaptureJavaProcessList() error = %v, wantErr %v", err, tt.wantErr)
			}
			c.GetProcListFromResponseBody()
			if len(c.GetResponseBody()) == 0 || strings.Contains(c.GetResponseBody(),"null") {
				t.Errorf("No Java processes found")
			}
			fmt.Println("Captured Java processes: ", c.GetResponseBody())
			slice := strings.Split(c.GetResponseBody(), "\n")
			for _, line := range slice {
				if strings.Contains(line, "UNNAMED") {
					t.Errorf("parsing mistake.. there's no process by that name")
				}
			}
			if len(slice) == 0 {
				t.Errorf("No Java processes found")
			}
			fmt.Println("Exit code: ", c.GetStatusCode())
		})
	}
}
