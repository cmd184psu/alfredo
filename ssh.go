package alfredo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type CrossCopyModeType int64

const (
	CCMVIAMEMORY CrossCopyModeType = iota
	CCMVIASHELL
	CCMTEMPFILE
)

const ssh_default_connection_timeout = 10

func (cc CrossCopyModeType) String() string {
	switch cc {
	case CCMTEMPFILE:
		return "temp"
	case CCMVIASHELL:
		return "shell"
	}
	return "memory"
}

func GetCCTypeOf(cc string) CrossCopyModeType {
	VerbosePrintln("getCCTypeOf(" + cc + ")")

	if strings.EqualFold(strings.ToLower(cc), CCMTEMPFILE.String()) {
		return CCMTEMPFILE
	}
	if strings.EqualFold(strings.ToLower(cc), CCMVIASHELL.String()) {
		return CCMVIASHELL
	}
	return CCMVIAMEMORY
}

type SSHStruct struct {
	Key            string `json:"key,omitempty"` //ssh key file, if empty, use ~/.ssh/id_rsa
	User           string `json:"user,omitempty"` //ssh user, if empty, use $USER
	Host           string `json:"host,omitempty"`
	capture        bool
	stdout         string
	stderr         string
	port           int
	RemoteDir      string `json:"remotedir,omitempty"` //remote directory to execute commands in, if empty, use home dir
	silent         bool
	exitCode       int
	ccmode         CrossCopyModeType
	ConnectTimeout int `json:"connecttimeout,omitempty"` //ssh -o ConnectTimeout=10
	request        string
	//parentExe      *ExecStruct
}

const (
	mkdir_p_fmt     = "mkdir -p %s"
	chown_r_fmt     = "chown -R %d:%d %s"
	SSH_DEFAULT_KEY = "~/.ssh/id_rsa"
)

func (s SSHStruct) GetSSHOptionsAsString() string {
	//does not work as expected, always leave the options blank for now
	// if s.ConnectTimeout != 0 {
	// 	return fmt.Sprintf("-o ConnectTimeout=%d ", s.ConnectTimeout)
	// }
	return ""
}

func (s SSHStruct) GetSSHCli() string {
	return fmt.Sprintf("ssh %s-i %s %s@%s", s.GetSSHOptionsAsString(), s.Key, s.User, s.Host)
}

func (s SSHStruct) parseSSHKey() (ssh.Signer, error) {
	var sign ssh.Signer
	rfp := ExpandTilde(s.Key)
	keyBytes, err := os.ReadFile(rfp)
	if err != nil {
		return sign, err
	}

	return ssh.ParsePrivateKey(keyBytes)
}

func (s *SSHStruct) CreateClientConfig() ssh.ClientConfig {
	var err error
	privateKey, err := s.parseSSHKey()
	if err != nil {
		panic(err.Error())
		//return ssh.ClientConfig{}
	}
	if s.ConnectTimeout == 0 {
		s.ConnectTimeout = ssh_default_connection_timeout
	}
	if len(s.User) == 0 {
		s.User = os.Getenv("USER")
	}
	return ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            s.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		}}
	//,
	//Timeout: time.Duration(s.ConnectTimeout)}
}

func (s SSHStruct) SecureDownload(remoteFilePath string, localFilePath string) error {
	// Parse the private key
	VerbosePrintf("remoteFilePath: %s", remoteFilePath)
	VerbosePrintf("localFilePath: %s", localFilePath)

	// Create an SSH client configuration
	config := s.CreateClientConfig()

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", s.Host+":22", &config)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// Create an SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Open the remote file
	remoteFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		VerbosePrintln("returning with err: " + err.Error())
		return err
	}
	defer remoteFile.Close()

	// Create or truncate the local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		VerbosePrintln("returning with err: " + err.Error())
		return err
	}
	defer localFile.Close()

	// Copy the remote file content to the local file
	_, err = io.Copy(localFile, remoteFile)
	return err
}

func (s *SSHStruct) SecureUploadContent(content []byte, remoteFilePath string) error {
	VerbosePrintln("BEGIN SecureUploadContent()")
	// Read the private key file
	// keyBytes, err := os.ReadFile(this.Key)
	// if err != nil {
	// 	VerbosePrintln("missing ssh key")
	// 	return err
	// }
	// Parse the private key
	//privateKey, err := s.parseSSHKey()
	// ..ParsePrivateKey(keyBytes)
	// if err != nil {
	// 	VerbosePrintln("ssh parse error")
	// 	return err
	// }
	VerbosePrintf("checking %s for existance...", remoteFilePath)
	if !s.RemoteFileExists(remoteFilePath) {
		VerbosePrintln("\tdoes not exist, touch it")
		if err := s.SecureRemoteExecution("touch " + remoteFilePath); err != nil {
			VerbosePrintln("\ttouch failed!")
			return err
		}
	}
	VerbosePrintln("\texists")
	VerbosePrintln("")
	VerbosePrintln("build ssh object")

	// Create an SSH client configuration
	config := s.CreateClientConfig()
	VerbosePrintf("dialing ssh with cli: %s", s.GetSSHCli())

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", s.Host+":22", &config)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	VerbosePrintln("\tabout to open sftp")

	// Create an SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		VerbosePrintf("\t\tunable to create sftp client with error: %s", err.Error())
		return err
	}
	defer sftpClient.Close()

	VerbosePrintln("\tcreating remote file: " + remoteFilePath)

	// Open the remote file

	VerbosePrintln("\tabout to create sftp")
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		VerbosePrintln("error creating remote file: " + remoteFilePath)
		return err
	}
	defer remoteFile.Close()
	VerbosePrintf("\tabout to write content: %d", len(content))

	w, err := remoteFile.Write(content)

	if err != nil {
		VerbosePrintln("err=" + err.Error())
	}
	if w == 0 {
		panic("zero bytes were written.. clearly, should not be the case")
	}
	// if ! this.silent {
	// 	fmt.Printf("wrote %d bytes\n", w)
	// }
	VerbosePrintln("END SecureUploadContent()")

	return err
}

func (s SSHStruct) SecureUpload(localFilePath string, remoteFilePath string) error {
	// Read the private key file
	// keyBytes, err := os.ReadFile(this.Key)
	// if err != nil {
	// 	return err
	// }

	// Parse the private key
	// privateKey, err := s.parseSSHKey()
	// //privateKey, err := ssh.ParsePrivateKey(keyBytes)
	// if err != nil {
	// 	return err
	// }

	// Create an SSH client configuration
	config := s.CreateClientConfig()

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", s.Host+":22", &config)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// Create an SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Open the remote file
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	// Create or truncate the local file
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Copy the remote file content to the local file
	//  write to, read from
	var w int64
	w, err = io.Copy(remoteFile, localFile)

	if err != nil {
		VerbosePrintln("err=" + err.Error())
	}
	if w == 0 {
		panic("zero bytes were written.. clearly, should be the case")
	}
	// if ! this.silent {
	// 	fmt.Printf("wrote %d bytes\n", w)
	// }

	return err
}

func (s SSHStruct) SecureDownloadAndSpin(remoteFilePath string, localFilePath string) error {
	localFilePath = ExpandTilde(localFilePath)
	return GoFuncAndSpin(s.SecureDownload, remoteFilePath, localFilePath)

	// var err error
	// var wg sync.WaitGroup

	// wg.Add(1)
	// sigChan := make(chan bool)
	// errorChan := make(chan error)
	// go func() {
	// 	defer wg.Done()
	// 	defer close(errorChan)
	// 	defer close(sigChan)

	// 	e := s.SecureDownload(remoteFilePath, localFilePath)
	// 	sigChan <- true
	// 	errorChan <- e
	// }()
	// if !GetQuiet() {
	// 	go Spinny(sigChan)
	// }
	// //errorRec = <-errorChan
	// err = <-errorChan
	// wg.Wait()
	// return err
}
func (s SSHStruct) SecureUploadAndSpin(localFilePath string, remoteFilePath string) error {
	localFilePath = ExpandTilde(localFilePath)
	return GoFuncAndSpin(s.SecureUpload, localFilePath, remoteFilePath)
	// var err error
	// var wg sync.WaitGroup

	// wg.Add(1)
	// sigChan := make(chan bool)
	// errorChan := make(chan error)
	// go func() {
	// 	defer wg.Done()
	// 	defer close(errorChan)
	// 	defer close(sigChan)

	// 	e := s.SecureUpload(localFilePath, remoteFilePath)
	// 	sigChan <- true
	// 	errorChan <- e
	// }()
	// if !GetQuiet() {
	// 	go Spinny(sigChan)
	// }
	// //errorRec = <-errorChan
	// err = <-errorChan
	// wg.Wait()
	// return err
}
func (ssh *SSHStruct) SetDefaults() {
	currentUser, err := user.Current()
	if err != nil {
		panic("Can't get current user")
	}
	if len(ssh.User) == 0 {
		ssh.User = currentUser.Name
	}
	if ssh.port == 0 {
		ssh.port = 22
	}
	if len(ssh.Key) == 0 {
		ssh.Key = filepath.Join(currentUser.HomeDir + "/.ssh/id_rsa")
	}
	if ssh.capture {
		ssh.stdout = ""
		ssh.stderr = ""
	}
}

func getExitCode(s string) int {
	//Process exited with status 2
	splits := strings.Split(s, " ")
	e, _ := strconv.Atoi(splits[len(splits)-1])
	return e
}

func (s *SSHStruct) RemoteTail(file string, lines int) ([]string, error) {
	return s.RemotePopenGrep("tail -n "+strconv.Itoa(lines)+" "+file, "", "")
}

func (s *SSHStruct) SecureRemoteExecution(cli string) error {
	if GetDryRun() {
		fmt.Printf("DRYRUN: %s\n", s.GetSSHCli()+" \""+cli+"\"")
		return nil
	}
	s.SetDefaults()
	// Replace with your remote server's SSH configuration
	if len(s.Host) == 0 {
		//		log.Fatalln("missing host")
		return fmt.Errorf("SSHStruct::SecureRemoteExecution::missing host")
	}

	// Read the private key
	// keyBytes, err := os.ReadFile(s.Key)
	// if err != nil {
	// 	log.Fatalf("Failed to read private key: %v", err)
	// }

	// Parse the private key
	//	signer, err := ssh.ParsePrivateKey(keyBytes)
	// signer, err := s.parseSSHKey()
	// if err != nil {
	// 	log.Fatalf("Failed to parse private key: %v", err)
	// }

	config := s.CreateClientConfig()

	VerbosePrintln(s.GetSSHCli() + " \"" + cli + "\"")

	// Connect to the remote server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.port), &config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Execute a remote command
	session, err := conn.NewSession()
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	if len(s.RemoteDir) > 0 {
		cli = "cd " + s.RemoteDir + " && " + cli
	}
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	if GetVerbose() {
		VerbosePrintln("---setting verbose in session ---")
		cli = "export VERBOSE=1; " + cli
		session.Setenv("VERBOSE", "1")
	}
	// else {
	// 	fmt.Println("---NOT setting verbose in session ---")
	// }

	//VerbosePrintln("SecureRemoteExecution: " + cli)
	//if this.capture {
	sessErr := session.Run(cli)
	stdoutBytes, _ := io.ReadAll(&stdoutBuf)
	stderrBytes, _ := io.ReadAll(&stderrBuf)

	if sessErr != nil {
		if !strings.HasPrefix(sessErr.Error(), "Process exited with") {
			//Process exited with status 2
			VerbosePrintln("sessErr: " + sessErr.Error())
			return sessErr
		}
		s.exitCode = getExitCode(sessErr.Error())
	} else {
		s.exitCode = 0
	}
	s.stdout = string(stdoutBytes)
	s.stderr = string(stderrBytes)

	if s.exitCode == 0 {
		return nil
	}
	VerbosePrintln("sessErr: " + sessErr.Error())
	VerbosePrintln("==== stderr ===")
	VerbosePrintln(s.stderr)
	VerbosePrintln("==== stdout ===")
	VerbosePrintln(s.stdout)
	return errors.New("ssh process exited with errors")
}

func (ssh SSHStruct) RemoteFileCount(sdirectoryPath string, prefix string, glob string) (int, error) {
	cli := GetFileFindCLI(sdirectoryPath, prefix, glob)
	if err := ssh.SecureRemoteExecution(cli); err != nil {
		return 0, err
	}
	result := strings.Split(strings.TrimSpace(ssh.GetBody()), "\n")
	// for l,r:= range result {
	// 	fmt.Printf("line: %d=%s\n",l, r)
	// }
	return len(result), nil
}

func (ssh SSHStruct) RemoteFindFiles(sdirectoryPath string, prefix string, glob string) ([]string, error) {
	cli := GetFileFindCLI(sdirectoryPath, prefix, glob)
	var result []string
	if err := ssh.SecureRemoteExecution(cli); err != nil {
		return result, err
	}
	result = strings.Split(ssh.GetBody(), "\n")
	return result[:len(result)-1], nil
}

func (ssh SSHStruct) RemoteRemoveFiles(sdirectoryPath string, prefix string, glob string) error {
	files, err := ssh.RemoteFindFiles(sdirectoryPath, prefix, glob)
	if err != nil {
		return err
	}
	for i := 0; i < len(files); i++ {
		if strings.EqualFold(strings.TrimSpace(files[i]), "") {
			return errors.New("blank file in list")
		}
		VerbosePrintln(fmt.Sprintf("removing file %s from host %s", files[i], ssh.Host))
		if err := ssh.RemoveRemoteFile(files[i]); err != nil {
			return err
		}
	}
	return nil
}

func (ssh SSHStruct) RenameRemoteFile(oldfile string, newfile string) error {
	return ssh.SecureRemoteExecution(fmt.Sprintf("mv -v %s %s", oldfile, newfile))
}

func (ssh SSHStruct) RemoveRemoteFile(file string) error {
	return ssh.SecureRemoteExecution(fmt.Sprintf("rm -vf %s", file))
}
func (ssh SSHStruct) RemoveRemoteDir(dir string) error {
	if strings.Contains(dir, "*") {
		panic("wildcard not allowed")
	}
	if len(dir) < 2 {
		panic("dir name too short")
	}
	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	return ssh.SecureRemoteExecution(fmt.Sprintf("rm -rf %q", dir))
}

func (s SSHStruct) GetRemoteHostname() (string, error) {
	if err := s.SecureRemoteExecution("hostname -s"); err != nil {
		return "", err
	}
	return strings.Trim(s.stdout, "\n"), nil
}

func (s *SSHStruct) RemoteExecuteAndSpin(cli string) error {
	return GoFuncAndSpin(s.SecureRemoteExecution, cli)
	// var err error
	// var wg sync.WaitGroup

	// wg.Add(1)
	// sigChan := make(chan bool)
	// errorChan := make(chan error)
	// go func() {
	// 	defer wg.Done()
	// 	defer close(errorChan)
	// 	defer close(sigChan)
	// 	var e error
	// 	s.capture = false
	// 	s.silent = true
	// 	e = s.SecureRemoteExecution(cli)
	// 	sigChan <- true
	// 	errorChan <- e
	// }()
	// if !GetQuiet() {
	// 	go Spinny(sigChan)
	// }
	// //errorRec = <-errorChan
	// err = <-errorChan
	// wg.Wait()
	// return err
}

// local, remote
func (this SSHStruct) UploadCLI(src string, tgt string) string {
	return fmt.Sprintf("Uploading %s to %s:%s", src, this.Host, tgt)
}

// remote, local
func (s SSHStruct) DownloadCLI(src string, tgt string) string {
	return fmt.Sprintf("Downloading from %s:%s %s", s.Host, src, tgt)
}

// return false on error or file doesn't exist (easy mode)
func (s SSHStruct) RemoteFileExists(path string) bool {
	err := s.SecureRemoteExecution("test -e " + path)
	if err != nil && strings.Contains(err.Error(), "i/o timeout") {
		fmt.Printf("WARNING: ssh connection timed out after %d sec(s)\n", s.ConnectTimeout)
	}
	return err == nil
}

func (s SSHStruct) BackgroundedRemoteExecute(cli string) (SSHStruct, error) {
	err := s.SecureRemoteExecution("nohup " + cli + " &")
	return s, err
}

func (s SSHStruct) WithCapture(c bool) SSHStruct {
	s.capture = c
	return s
}
func (s *SSHStruct) SetCapture(c bool) SSHStruct {
	s.capture = c
	return *s
}

func (s SSHStruct) GetRequest() string {
	return s.request
}
func (s *SSHStruct) SetRequest(r string) {
	s.request = r
}
func (s SSHStruct) WithRequest(r string) SSHStruct {
	s.SetRequest(r)
	return s
}

func (s SSHStruct) WithSilent(c bool) SSHStruct {
	s.silent = c
	return s
}
func (s SSHStruct) WithKey(k string) SSHStruct {
	s.Key = k
	return s
}

func (s *SSHStruct) SetHost(h string) SSHStruct {
	s.Host = h
	return *s
}
func (s *SSHStruct) SetRemoteDir(r string) SSHStruct {
	s.RemoteDir = r
	return *s
}
func (s SSHStruct) WithHost(h string) SSHStruct {
	s.Host = h
	return s
}
func (s SSHStruct) WithUser(u string) SSHStruct {
	s.User = u
	return s
}
func (s SSHStruct) WithRemoteDir(rd string) SSHStruct {
	s.RemoteDir = rd
	return s
}

func (s SSHStruct) GetStdout() string {
	return s.stdout
}
func (s SSHStruct) GetStderr() string {
	return s.stderr
}
func (s SSHStruct) GetBody() string {
	return s.GetStdout()
}

func (s SSHStruct) GetBodyBytes() []byte {
	return []byte(s.stdout)
}

func (s SSHStruct) GetRemoteDir() string {
	return s.RemoteDir
}

func (s SSHStruct) MkdirAll(dir string) error {
	return s.SecureRemoteExecution(fmt.Sprintf(mkdir_p_fmt, dir))
}

func (s SSHStruct) Chown(uid int, gid int, path string) error {
	return s.SecureRemoteExecution(fmt.Sprintf(chown_r_fmt, uid, gid, path))
}

func (s *SSHStruct) Execute(cli string) error {
	if len(s.request) > 0 {
		return s.SecureRemotePipeExecution([]byte(s.request), cli)
	}
	return s.SecureRemoteExecution(cli)
}

func (s *SSHStruct) SecureRemotePipeExecution(content []byte, cli string) error {
	if GetDryRun() {
		fmt.Printf("DRYRUN: send %s over pipe to %s\n", string(content), cli)
		return nil
	}

	VerbosePrintln("!!! SecureRemotePipeExecution(...) !!!")
	// SSH configuration
	// keyBytes, err := os.ReadFile(s.Key)
	// if err != nil {
	// 	return err
	// }

	VerbosePrintln("parsing key, in memory")
	// Parse the private key
	//privateKey, err := ssh.ParsePrivateKey(keyBytes)
	// privateKey, err := s.parseSSHKey()
	// if err != nil {
	// 	return err
	// }

	VerbosePrintln("creating ssh config")
	config := s.CreateClientConfig()

	VerbosePrintln("ssh dial up to " + s.Host + ":22")
	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", s.Host+":22", &config)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	VerbosePrintln("establishing new session")
	// Open a new session
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Start the remote command and get pipes for its stdin and stdout
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	//defer stdout.Close()

	// Run the remote command
	err = session.Start(cli)
	if err != nil {
		return err
	}

	// Write the []byte to the standard input of the remote command
	_, err = stdin.Write(content)
	if err != nil {
		return err
	}

	// Close the standard input of the remote command
	stdin.Close()

	// Read the output of the remote command
	var outputBuffer bytes.Buffer
	_, err = io.Copy(&outputBuffer, stdout)
	if err != nil {
		return err
	}

	// Wait for the remote command to finish
	err = session.Wait()
	if err != nil {
		return err
	}

	// Print the output
	//fmt.Println("Remote command output:", outputBuffer.String())
	VerbosePrintln("acquire body from session")
	s.stdout = outputBuffer.String()
	return nil
}

func (s SSHStruct) NotConfigured() bool {
	return len(s.Key) == 0 || len(s.Host) == 0
}

const dd_cli_fmt = "dd if=%s of=%s bs=1k count=1 seek=%d"

func (s SSHStruct) WriteSparseFile(f string, sizeMin int, sizeMax int, r int) error {
	minSize := sizeMin * 1024
	return s.SecureRemoteExecution(fmt.Sprintf(dd_cli_fmt, "/dev/random", f, rand.Intn(sizeMax*1024-minSize+1)+minSize))
}

func (s *SSHStruct) RemoveDirAndContent(d string) error {
	VerbosePrintln("BEGIN RemoveDirAndContent(" + d + ")")
	if len(d) > 2 && !strings.Contains(d, "*") && !strings.Contains(d, ".") {
		VerbosePrintln("rm -rfv " + d)
		VerbosePrintln("END RemoveDirAndContent(" + d + ")")
		return s.RemoteExecuteAndSpin("rm -rfv " + d)
		//return nil
	}
	return errors.New("removedir request did not pass requirements")
}

func (s *SSHStruct) RemoteReadFile(f string) error {
	return s.RemoteExecuteAndSpin(fmt.Sprintf("cat %s", f))
}

func (s *SSHStruct) GetExitCode() int {
	return s.exitCode
}

func (srcssh *SSHStruct) CrossCopy(srcFile string, tgtssh SSHStruct, tgtFile string) error {

	switch srcssh.ccmode {
	case CCMTEMPFILE:
		VerbosePrintln("crossCopyvia tempfile")
		return srcssh.crossCopyViaTempFile(srcFile, tgtssh, tgtFile)
	case CCMVIASHELL:
		VerbosePrintln("crossCopyvia shell")
		return srcssh.crossCopyViaShell(srcFile, tgtssh, tgtFile)
	}
	VerbosePrintln("crossCopyvia memory")
	return srcssh.crossCopyInMemory(srcFile, tgtssh, tgtFile)
}

func (srcssh *SSHStruct) crossCopyViaTempFile(srcFile string, tgtssh SSHStruct, tgtFile string) error {
	tempfile := ExpandTilde("~/TMP-blob")
	if err := srcssh.SecureDownloadAndSpin(srcFile, tempfile); err != nil {
		return err
	}

	// if err := tgtssh.SecureUploadContent(srcssh.GetBodyBytes(), tgtFile); err != nil {
	// 	return err
	// }
	if err := tgtssh.SecureUploadAndSpin(tempfile, tgtFile); err != nil {
		return err
	}
	return RemoveFile(tempfile)
}

func (srcssh *SSHStruct) crossCopyInMemory(srcFile string, tgtssh SSHStruct, tgtFile string) error {
	panicOnFail = true
	VerbosePrintf("BEGIN ssh::crossCopyInMemory(%s,%s(ssh host),%s)", srcFile, tgtssh.Host, tgtFile)
	VerbosePrintf("remote read file %s from %s", srcFile, srcssh.Host)
	if err := srcssh.RemoteReadFile(srcFile); err != nil {
		VerbosePrintln("failed on read")
		return PanicError(err.Error())
	}

	VerbosePrintf("remote write content of size %s to remote file %s:%s", HumanReadableBigNumber(int64(len(srcssh.GetBodyBytes()))), tgtssh.Host, tgtFile)
	if err := tgtssh.SecureUploadContent2(srcssh.GetBodyBytes(), tgtFile); err != nil {
		VerbosePrintln("failed on write")
		return PanicError(err.Error())
	}
	VerbosePrintf("END ssh::crossCopyInMemory(%s,%s(ssh host),%s)", srcFile, tgtssh.Host, tgtFile)
	return nil
}

func (srcssh *SSHStruct) CrossCopyCLI(srcFile string, tgtssh SSHStruct, tgtFile string) string {
	return fmt.Sprintf("scp -3 -o IdentityFile=%s -o IdentityFile=%s %s@%s:%s %s@%s:%s",
		srcssh.Key,
		tgtssh.Key,
		srcssh.User, srcssh.Host, srcFile,
		tgtssh.User, tgtssh.Host, tgtFile)
}

func (srcssh *SSHStruct) crossCopyViaShell(srcFile string, tgtssh SSHStruct, tgtFile string) error {
	cli := srcssh.CrossCopyCLI(srcFile, tgtssh, tgtFile)

	var exe ExecStruct
	exe.Init().
		WithMainExecFunc(System3toCapturedString, cli).
		WithSpinny(true).
		WithCapture(true).
		WithDirectory(".")
	return exe.Execute()
}

const (
	git_revision = "rev-parse --short HEAD"
	git_branch   = "rev-parse --abbrev-ref HEAD"
	VERSION_FILE = "VERSION"
	RELEASE_FILE = "RELEASE"
)

func (s *SSHStruct) RemoteGitRev() string {
	return s.RemoteGit(git_revision)
}
func (s *SSHStruct) RemoteGitBranch() string {
	return s.RemoteGit(git_branch)
}
func (s *SSHStruct) RemoteGit(gitargs string) string {
	if err := s.SecureRemoteExecution(fmt.Sprintf("git %s", gitargs)); err != nil {
		panic(err.Error())
	}
	return strings.TrimSpace(s.GetBody())
}

func (s *SSHStruct) RemoteGetVersion() string {
	if err := s.RemoteReadFile(VERSION_FILE); err != nil {
		panic(err.Error())
	}
	return strings.TrimSpace(s.GetBody())
}

func (s *SSHStruct) RemoteGetRelease() int {
	if err := s.RemoteReadFile(RELEASE_FILE); err != nil {
		panic(err.Error())
	}
	r, _ := strconv.Atoi(strings.TrimSpace(s.GetBody()))
	return r
}

const (
	//	go_build_cli = "go build -ldflags -X 'main.GitRevision=\\\"%s\\\" -X main.GitBranch=\\\"%s\\\" -X main.GitVersion=\\\"%s\\\" -X main.GitTimestamp=\\\"%s\\\"' -o %s"
	go_build_cli = "go build -ldflags '-X main.GitRevision=\"%s\" -X main.GitBranch=\"%s\" -X main.GitVersion=\"%s\" -X main.GitTimestamp=\"%s\"' -o %s"
)

func (s SSHStruct) GenerateRemoteGoBuildCLI(binary string) string {
	VerbosePrintf("formated time 1 (now): %s", GetFormattedTime1())

	return fmt.Sprintf(go_build_cli,
		s.RemoteGitRev(),
		s.RemoteGitBranch(),
		s.RemoteGetVersion(),
		GetFormattedTime1(),
		binary)
}

func (s *SSHStruct) SecureUploadContent2(content []byte, remoteFilePath string) error {
	VerbosePrintln("BEGIN SecureUploadContent2()")
	VerbosePrintf("checking %s for existance...", remoteFilePath)
	if !s.RemoteFileExists(remoteFilePath) {
		VerbosePrintln("\tdoes not exist, touch it")
		if err := s.SecureRemoteExecution("touch " + remoteFilePath); err != nil {
			VerbosePrintln("\ttouch failed!")
			return err
		}
	}
	cli := fmt.Sprintf("cat > %s", remoteFilePath)
	if err := s.SecureRemotePipeExecution(content, cli); err != nil {
		return err
	}

	VerbosePrintln("END SecureUploadContent2()")
	return nil
}

func (s *SSHStruct) RemotePopenGrep(cli string, musthave string, mustnothave string) ([]string, error) {
	VerbosePrintf("BEGIN RemotePopenGrep(%s,%s,%s)", cli, musthave, mustnothave)
	var mh, mnh string
	if len(musthave) > 0 {
		mh = fmt.Sprintf(" | grep %s", musthave)
	}
	if len(mustnothave) > 0 {
		mnh = fmt.Sprintf(" | grep -v %s", mustnothave)
	}
	VerbosePrintf("\tcli: %s", s.GetSSHCli())
	if err := s.SecureRemoteExecution(fmt.Sprintf("%s%s%s", cli, mh, mnh)); err != nil {
		return []string{}, err
	}
	VerbosePrintf("END RemotePopenGrep(%s,%s,%s)", cli, musthave, mustnothave)
	return strings.Split(strings.TrimSpace(s.GetBody()), "\n"), nil
}

func (s *SSHStruct) RemoteJPS() ([]string, error) {
	return s.RemotePopenGrep("ps aux", "java", "grep")
}

const pid_status_format = "/proc/%d/status"
const pid_cmdline_format = "/proc/%d/cmdline"

func (s SSHStruct) RemoteGetThreadCount(pid int) (int, error) {
	if pid == -1 {
		return 0, nil
	}
	if err := s.RemoteReadFile(fmt.Sprintf(pid_status_format, pid)); err != nil {
		return 0, err
	}
	return ParseThreadCount(s.GetBody())
}

func (s SSHStruct) RemoteGetArgsFromPid(pid int) ([]string, error) {
	if pid == -1 {
		return []string{}, nil
	}
	if err := s.RemoteReadFile(fmt.Sprintf(pid_cmdline_format, pid)); err != nil {
		return []string{}, err
	}
	return strings.Split(s.GetBody(), "\x00"), nil
}

func (s SSHStruct) RemoteCapturePid(jvm string, hint string) (int, error) {
	VerbosePrintln("inside CapturePid(" + jvm + ")")
	lines, err := s.RemoteJPS()
	if err != nil {
		return 0, err
	}
	jlist := JPSStructListToIntList(SlicetoJPSStruct(lines, hint), jvm)
	if len(jlist) == 0 {
		return 0, fmt.Errorf(no_processes_found)
	}
	if len(jlist) > 1 {
		VerbosePrintln("multiple processes found, returning first")
	}
	return jlist[0], nil
}

func (s SSHStruct) ReportHammerResults(results []bool, e error) error {
	successCount := 0
	failureCount := 0
	failedTasks := []int{}

	for i, success := range results {
		if success {
			successCount++
		} else {
			failureCount++
			failedTasks = append(failedTasks, i)
		}
	}

	fmt.Printf("Total tasks: %d\n", len(results))
	fmt.Printf("Successful tasks: %d\n", successCount)
	fmt.Printf("Failed tasks: %d\n", failureCount)

	if len(failedTasks) > 0 {
		fmt.Printf("Failed task IDs: %v\n", failedTasks)
	}
	return e
}

func (s SSHStruct) HammerTest() error {
	if len(s.Key) == 0 {
		return fmt.Errorf("missing ssh key")
	}
	if len(s.Host) == 0 {
		return fmt.Errorf("missing ssh host")
	}

	x := 18 // Number of times to run the function concurrently
	results := Concurrent(s.RemoteFileExists, "/bin/true", x)

	// Display and analyze results
	fmt.Println("Function results:", results)
	successes := 0
	failures := 0
	for _, res := range results {
		if res {
			successes++
		} else {
			failures++
		}
	}
	fmt.Printf("Successes: %d, Failures: %d\n", successes, failures)
	return nil
}

func (s SSHStruct) GetRemoteFileHash(path string) (string, error) {
	if err := s.SecureRemoteExecution(fmt.Sprintf("md5sum %s", path)); err != nil {
		return "error", err
	}
	return strings.Trim(s.stdout, "\n"), nil
}

func (s *SSHStruct) Rsync(source string, target string) error {
	return s.RsyncWithSwitches("-avz", source, target)
}

func (s *SSHStruct) RsyncWithSwitches(switches string, source string, target string) error {
	sshCmd := "ssh -i " + ExpandTilde(s.Key)

	// Construct the rsync command
	cmd := exec.Command("rsync",
		switches,
		"-e", sshCmd,
		source,
		fmt.Sprintf("%s@%s:%s", s.User, s.Host, target),
	)

	// Run the command and capture output
	output, err := cmd.CombinedOutput()
	s.stdout = string(output)

	return err
}

func (s *SSHStruct) GetLastModifiedTime(remoteFile string) (time.Time, error) {
	// Execute the command to get the last modified time
	if err := s.SecureRemoteExecution(fmt.Sprintf("stat -c %%Y %s", remoteFile)); err != nil {
		return time.Time{}, err
	}
	epochSeconds, err := strconv.ParseInt(strings.TrimSpace(s.GetBody()), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(epochSeconds, 0), nil
}

func (s *SSHStruct) GetRemoteFileSize(remoteFile string) (int64, error) {
	// Execute the command to get the file size in bytes
	if err := s.SecureRemoteExecution(fmt.Sprintf("stat -c %%s %s", remoteFile)); err != nil {
		return 0, err
	}
	sizeStr := strings.TrimSpace(s.GetBody())
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (s *SSHStruct) SyncFileWithRemote(localFile string, remoteFile string, hashValidation bool, createDirectories bool) error {
	// Check if the SSH connection is configured
	localFile = ExpandTilde(localFile)
	if s.NotConfigured() {
		return fmt.Errorf("SSH connection not configured")
	}
	skipUpload := false
	skipDownload := false

	hashLocal := ""
	hashRemote := ""

	// Check if the local file exists
	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		VerbosePrintf("local file does not exist: %s", localFile)
		skipUpload = true
		if createDirectories {
			fmt.Println("Creating directories on local host...")
			ex := NewCLIExecutor()
			ex.WithCommand(fmt.Sprintf("mkdir -p %s", filepath.Dir(localFile)))
			if err := ex.DumpOutput().Execute(); err != nil {
				fmt.Printf("Unable to create local directories due to err: %s\n", err)
				return err
			}
		}
	}
	// Check if the remote file exists

	if !s.RemoteFileExists(remoteFile) {
		VerbosePrintf("remote file does not exist: %s", remoteFile)
		skipDownload = true
		if createDirectories {
			fmt.Println("Creating directories on remote host...")
			if err := s.RemoteExecuteAndSpin(fmt.Sprintf("mkdir -p %s", filepath.Dir(remoteFile))); err != nil {
				return err
			}
		}
	}
	// if err := s.SecureRemoteExecution(fmt.Sprintf("test -e %s", remoteFile)); err != nil {
	// 	if strings.Contains(err.Error(), "No such file or directory") {
	// 		// Remote file does not exist, proceed with upload
	// 		VerbosePrintln("Remote file does not exist, proceeding with upload")
	// 		skipDownload = true
	// 	} else if strings.Contains(err.Error(), "Permission denied") {
	// 		// Permission denied error, handle accordingly
	// 		return fmt.Errorf("permission denied for remote file: %s", remoteFile)
	// 	} else if strings.Contains(err.Error(), "Connection refused") {
	// 		// SSH connection refused
	// 		return fmt.Errorf("SSH connection refused")
	// 	} else if strings.Contains(err.Error(), "i/o timeout") {
	// 		// SSH connection timed out
	// 		return fmt.Errorf("SSH connection timed out")
	// 	} else {
	// 		// Some other error occurred
	// 		return fmt.Errorf("failed to check remote file: %v", err)
	// 	}
	// }

	//at this point we've determined if the local file exists or not
	// we've determined if the remote file exists or not
	// we've determine that the remote hots is at least reachable
	var localLastModified, remoteLastModified time.Time
	var err error
	if skipDownload {
		remoteLastModified = time.Time{}
	} else {
		//this means the remote file does exist
		remoteLastModified, err = s.GetLastModifiedTime(remoteFile)
		if err != nil {
			// Handle error
			return fmt.Errorf("failed to get remote file last modified time: %v", err)
		}
		if hashValidation {
			ex := NewCLIExecutor()
			ex.WithSSHStruct(*s)
			hashRemote = ex.HashFile(remoteFile)
		}
	}

	if skipUpload {
		localLastModified = time.Time{}
	} else {
		localLastModified, err = GetLastModifiedTime(localFile)
		if err != nil {
			// Handle error
			return fmt.Errorf("failed to parse local file last modified time: %v", err)
		}
		if hashValidation {
			ex := NewCLIExecutor()

			VerbosePrintln("just before hashing local file")
			hashLocal = ex.HashFile(localFile)
			VerbosePrintln("just after hashing local file")
			// if err != nil {
			// 	return fmt.Errorf("failed to get local file hash: %v", err)
			// }
		}
	}
	if hashValidation {
		if strings.EqualFold(hashLocal, hashRemote) {
			VerbosePrintln("Hashes match, files are already in sync")
			return nil
		} else {
			VerbosePrintln("Hashes do not match, proceeding with upload")

		}
	}
	if localLastModified.After(remoteLastModified) {
		// Local file is newer, upload it
		VerbosePrintln("Local file is newer, proceeding with upload")
		if err := s.SecureUploadAndSpin(localFile, remoteFile); err != nil {
			return fmt.Errorf("failed to upload local file: %v", err)
		}
		if hashValidation {
			ex := NewCLIExecutor()
			ex.WithSSHStruct(*s)
			// ex.HashFilePre(remoteFile)
			// VerbosePrintln("just before hashing remote file")
			// VerbosePrintf("here is the cli: %q", ex.GetCli())
			// //			hashRemote = ex.HashFile(remoteFile)

			// if err := ex.Execute(); err != nil {
			// 	return fmt.Errorf("failed to get remote file hash: %v", err)
			// }
			// hashRemote = ex.HashFilePost()
			hashRemote = ex.HashFile(remoteFile)
			VerbosePrintln("just before hashing remote file")
		}

	} else {
		// Remote file is newer, download it
		VerbosePrintln("Remote file is newer, proceeding with download")
		VerbosePrintf("\tdownloading %s ===> %s", remoteFile, localFile)
		if err := s.SecureDownloadAndSpin(remoteFile, localFile); err != nil {
			return fmt.Errorf("failed to download remote file: %v", err)
		}
		if hashValidation {
			ex := NewCLIExecutor()

			hashLocal = ex.HashFile(localFile)
		}
	}

	if hashValidation && !strings.EqualFold(hashLocal, hashRemote) {
		fmt.Printf("local hash: %q\n", hashLocal)
		fmt.Printf("remote hash: %q\n", hashRemote)
		return fmt.Errorf("file hashes do not match after transfer")
	}

	// Upload the file to the remote server
	// if err := s.SecureUpload(localFile, remoteFile); err != nil {
	// 	return fmt.Errorf("failed to upload file to remote server: %v", err)
	// }

	return nil
}

func (s *SSHStruct) Load(fileName string) error {
	return ReadStructFromJSONFile(fileName, s)
}
