package alfredo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cmd184psu/fs-tools/fstools-gomod"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHStruct struct {
	Key       string `json:"key"`
	User      string `json:"user"`
	Host      string `json:"host"`
	capture   bool
	stdout    string
	stderr    string
	port      int
	remoteDir string
	silent    bool
	exitCode  int
}

const (
	mkdir_p_fmt = "mkdir -p %s"
	chown_r_fmt = "chown -R %d:%d %s"
)

func (s SSHStruct) GetSSHCli() string {
	return "ssh -i " + s.Key + " " + s.User + "@" + s.Host
}

func (this SSHStruct) SecureDownload(remoteFilePath string, localFilePath string) error {
	// Read the private key file
	keyBytes, err := os.ReadFile(this.Key)
	if err != nil {
		return err
	}

	// Parse the private key
	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}

	// Create an SSH client configuration
	config := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            this.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
	}

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", this.Host+":22", config)
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

func (this *SSHStruct) SecureUploadContent(content []byte, remoteFilePath string) error {
	// Read the private key file
	keyBytes, err := os.ReadFile(this.Key)
	if err != nil {
		VerbosePrintln("missing ssh key")
		return err
	}
	if !this.RemoteFileExists(remoteFilePath) {
		if err := this.SecureRemoteExecution("touch " + remoteFilePath); err != nil {
			return err
		}
	}
	// Parse the private key
	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		VerbosePrintln("ssh parse error")
		return err
	}

	// Create an SSH client configuration
	config := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            this.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
	}

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", this.Host+":22", config)
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

	VerbosePrintln("creating remote file: " + remoteFilePath)

	// Open the remote file
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		VerbosePrintln("error creating remote file: " + remoteFilePath)
		return err
	}
	defer remoteFile.Close()

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

	return err
}

func (this SSHStruct) SecureUpload(localFilePath string, remoteFilePath string) error {
	// Read the private key file
	keyBytes, err := os.ReadFile(this.Key)
	if err != nil {
		return err
	}

	// Parse the private key
	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}

	// Create an SSH client configuration
	config := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            this.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
	}

	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", this.Host+":22", config)
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

func (this SSHStruct) SecureDownloadAndSpin(remoteFilePath string, localFilePath string) error {
	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)

		e := this.SecureDownload(remoteFilePath, localFilePath)
		sigChan <- true
		errorChan <- e
	}()
	go Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return err
}
func (this SSHStruct) SecureUploadAndSpin(localFilePath string, remoteFilePath string) error {
	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)

		e := this.SecureUpload(localFilePath, remoteFilePath)
		sigChan <- true
		errorChan <- e
	}()
	go fstools.Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return err
}
func (this *SSHStruct) SetDefaults() {
	currentUser, err := user.Current()
	if err != nil {
		panic("Can't get current user")
	}
	if len(this.User) == 0 {
		this.User = currentUser.Name
	}
	if this.port == 0 {
		this.port = 22
	}
	if len(this.Key) == 0 {
		this.Key = filepath.Join(currentUser.HomeDir + "/.ssh/id_rsa")
	}
	if this.capture {
		this.stdout = ""
		this.stderr = ""
	}
}

func getExitCode(s string) int {
	//Process exited with status 2
	splits := strings.Split(s, " ")
	e, _ := strconv.Atoi(splits[len(splits)-1])
	return e
}
func (s *SSHStruct) SecureRemoteExecution(cli string) error {
	s.SetDefaults()
	// Replace with your remote server's SSH configuration
	if len(s.Host) == 0 {
		//		log.Fatalln("missing host")
		panic("SecureRemoteExecution::missing host")
	}

	// Read the private key
	keyBytes, err := os.ReadFile(s.Key)
	if err != nil {
		log.Fatalf("Failed to read private key: %v", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	VerbosePrintln(s.GetSSHCli() + " \"" + cli + "\"")

	// Connect to the remote server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.port), config)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Execute a remote command
	session, err := conn.NewSession()
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	if len(s.remoteDir) > 0 {
		cli = "cd " + s.remoteDir + " && " + cli
	}
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
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

	VerbosePrintln("SecureRemoteExecution: " + cli)
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

func (s SSHStruct) GetRemoteHostname() (string, error) {
	if err := s.SecureRemoteExecution("hostname -s"); err != nil {
		return "", err
	}
	return strings.Trim(s.stdout, "\n"), nil
}

func (s *SSHStruct) RemoteExecuteAndSpin(cli string) error {
	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)
		var e error
		s.capture = false
		s.silent = true
		e = s.SecureRemoteExecution(cli)
		sigChan <- true
		errorChan <- e
	}()
	go Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return err
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
	s.remoteDir = r
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
	s.remoteDir = rd
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

func (s SSHStruct) GetRemoteDir() string {
	return s.remoteDir
}

func (s SSHStruct) MkdirAll(dir string) error {
	return s.SecureRemoteExecution(fmt.Sprintf(mkdir_p_fmt, dir))
}

func (s SSHStruct) Chown(uid int, gid int, path string) error {
	return s.SecureRemoteExecution(fmt.Sprintf(chown_r_fmt, uid, gid, path))
}

func (s *SSHStruct) SecureRemotePipeExecution(content []byte, cli string) error {
	VerbosePrintln("!!! SecureRemotePipeExecutetion(...) !!!")
	// SSH configuration
	keyBytes, err := os.ReadFile(s.Key)
	if err != nil {
		return err
	}

	VerbosePrintln("parsing key, in memory")
	// Parse the private key
	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}

	VerbosePrintln("creating ssh config")
	config := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            s.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
	}

	VerbosePrintln("ssh dial up to " + s.Host + ":22")
	// Connect to the remote host
	sshClient, err := ssh.Dial("tcp", s.Host+":22", config)
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
	return len(s.Key) == 0 || len(s.Host) == 0 || len(s.User) == 0
}

const dd_cli_fmt = "dd if=%s of=%s bs=1k count=1 seek=%d"

func (s SSHStruct) WriteSparseFile(f string, sizeMin int, sizeMax int, r int) error {
	minSize := sizeMin * 1024
	return s.SecureRemoteExecution(fmt.Sprintf(dd_cli_fmt, "/dev/random", f, rand.Intn(sizeMax*1024-minSize+1)+minSize))
}

func (s SSHStruct) RemoveDirAndContent(d string) error {
	VerbosePrintln("BEGIN RemoveDirAndContent(" + d + ")")
	if len(d) > 2 && !strings.Contains(d, "*") && !strings.Contains(d, ".") {
		VerbosePrintln("rm -rfv " + d)
		VerbosePrintln("END RemoveDirAndContent(" + d + ")")
		return s.RemoteExecuteAndSpin("rm -rfv " + d)
		//return nil
	}
	return errors.New("removedir request did not pass requirements")
}
