package alfredo

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
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
	body      string
	port      int
	remoteDir string
	silent    bool
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
		return err
	}
	defer remoteFile.Close()

	// Create or truncate the local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Copy the remote file content to the local file
	_, err = io.Copy(localFile, remoteFile)
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
	} else {
		fmt.Printf("wrote %d bytes\n", w)
	}

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
		this.body = ""
	}
}
func (this *SSHStruct) SecureRemoteExecution(cli string) error {
	this.SetDefaults()
	// Replace with your remote server's SSH configuration
	if len(this.Host) == 0 {
		//		log.Fatalln("missing host")
		panic("missing host")
	}

	// Read the private key
	keyBytes, err := os.ReadFile(this.Key)
	if err != nil {
		log.Fatalf("Failed to read private key: %v", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: this.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	VerbosePrintln(this.GetSSHCli() + " \"" + cli + "\"")

	// Connect to the remote server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", this.Host, this.port), config)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Execute a remote command
	session, err := conn.NewSession()

	if len(this.remoteDir) > 0 {
		cli = "cd " + this.remoteDir + " && " + cli
	}
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	VerbosePrintln("SecureRemoteExecution: " + cli)
	//if this.capture {
	barray, sessErr := session.CombinedOutput(cli)
	if sessErr != nil {
		VerbosePrintln("sessErr: " + sessErr.Error())
		return sessErr
	}
	this.body = string(barray)
	return err
}

func (s SSHStruct) GetRemoteHostname() (string, error) {
	var err error
	err = s.SecureRemoteExecution("hostname -s")
	if err != nil {
		return "", err
	}
	return strings.Trim(s.body, "\n"), nil
}

func (this SSHStruct) RemoteExecuteAndSpin(cli string) (SSHStruct, error) {
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
		this.capture = false
		this.silent = true
		e = this.SecureRemoteExecution(cli)
		sigChan <- true
		errorChan <- e
	}()
	go Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return this, err
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

func (s SSHStruct) GetBody() string {
	return s.body
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
