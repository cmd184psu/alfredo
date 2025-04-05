package alfredo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const private_ssh_key_default = "%s/.ssh/id_rsa"

var ssh_key = ""

func GetPrivateSSHKey() string {
	if strings.EqualFold(ssh_key, "") {
		ssh_key = fmt.Sprintf(private_ssh_key_default, os.Getenv("HOME"))
	}
	return ssh_key
}

func SetPrivateSSHKey(newkey string) {
	ssh_key = newkey
}

func Touch(fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err == nil {
			defer file.Close()
		}
		return err
	} else {
		currentTime := time.Now().Local()
		return os.Chtimes(fileName, currentTime, currentTime)
	}
}

func System3toCapturedString(s *string, cmd string) error {
	if GetDryRun() {
		fmt.Printf("DRYRUN: %s\n", cmd)
		return nil
	}
	VerbosePrintln(cmd)
	for strings.Contains(cmd, "  ") {
		cmd = strings.ReplaceAll(strings.TrimSpace(cmd), "  ", " ")
	}
	arglist := strings.Split(cmd, " ")
	arglist2 := make([]string, 0)

	st := 0
	dir := ""
	if len(arglist) > 3 && strings.EqualFold(arglist[0], "cd") && strings.EqualFold(arglist[2], "&&") {
		dir = arglist[1]
		st = 3
	}

	for i := st; i < len(arglist); i++ {
		//		VerbosePrintf("arglist[%d]=%s", i, arglist[i]])
		if strings.HasPrefix(arglist[i], "\"") {
			for j := i; j < len(arglist); j++ {
				arglist2 = append(arglist2, arglist[j])
			}

			arglist = arglist[:i]
			arglist = append(arglist, TrimQuotes(strings.Join(arglist2, " ")))

			break
		}
	}

	for i := 0; i < len(arglist); i++ {
		VerbosePrintln(fmt.Sprintf("arg[%d]=%s\n", i, arglist[i]))
	}

	var b bytes.Buffer
	VerbosePrintf("cmd=%s", arglist[st])
	var runme *exec.Cmd
	if st+1 >= len(arglist) {
		runme = exec.Command(arglist[st])

	} else {
		runme = exec.Command(arglist[st], arglist[st+1:]...)
	}
	if len(dir) > 0 {
		VerbosePrintf("dir=%s", dir)
		runme.Dir = dir
	}
	// runme.Env = os.Environ()
	// VerbosePrintln(strings.Join(runme.Env, "\n"))
	err := Popen3(&b, runme)

	*s = b.String()
	return err
}

func System3(cmd string) error {
	var s string
	if err := System3toCapturedString(&s, cmd); err != nil {
		return err
	}
	_, err := io.Copy(os.Stdout, strings.NewReader(s))
	return err
}

// was "readlines"
func Grep(path string, musthave string, mustnothave string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if (musthave == "" || strings.Contains(scanner.Text(), musthave)) &&
			(mustnothave == "" || !strings.Contains(scanner.Text(), mustnothave)) {
			lines = append(lines, scanner.Text())
		}
	}
	return lines, scanner.Err()
}

func ReadFileToSlice(filename string, force bool) ([]string, error) {
	if b, err := FileExists(filename); err != nil || !b {
		if err != nil && !force {
			//file existance test fails entirely ( doesn't mean file doesn't exist )
			return make([]string, 0), err

		}
		if !b && !force {
			//file does not exist, but we want to throw an error indicating as such
			return make([]string, 0), errors.New("file not found")
		}
		//ignore the fact that the file doesn't exist and return an empty array, quietly
		return make([]string, 0), nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func WriteSliceToFile(filename string, content []string) error {
	return WriteStringToFile(filename, strings.Join(content, "\n"))
}

func WriteStringToFile(filename string, content string) error {
	f, err := os.OpenFile(filename+".tmp", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		panic(err)
	}

	return MoveFile(filename+".tmp", filename)
}
func AppendStringToFile(filename string, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		panic(err)
	}
	return err
}

// formerly Execute
func Popen3(output_buffer *bytes.Buffer, stack ...*exec.Cmd) (err error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", stack)
		return nil
	}
	var error_buffer bytes.Buffer
	pipe_stack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &error_buffer
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = output_buffer
	stack[i].Stderr = &error_buffer

	// if err := call(stack, pipe_stack); err != nil {
	// 	log.Fatalln(string(error_buffer.Bytes()), err)
	// }
	// return err
	return call(stack, pipe_stack)
}

func call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if stack[0].Process == nil {
		if err = stack[0].Start(); err != nil {
			return err
		}
	}
	if len(stack) > 1 {
		if err = stack[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = call(stack[1:], pipes[1:])
			}
		}()
	}
	return stack[0].Wait()
}

func FileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func PopentoString(cmd string) (string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd)
		return "", nil
	}
	arglist := strings.Split(cmd, " ")
	//app:=arglist[0]

	var b bytes.Buffer
	if err := Popen3(&b,
		exec.Command(arglist[0], arglist[1:]...),
		exec.Command("head", "-1"),
	); err != nil {
		log.Fatalln(err)
	}
	return b.String(), nil
}

func PopentoStringAwk(cmd string, awk int) (string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd)
		return "", nil
	}
	arglist := strings.Split(cmd, " ")
	//app:=arglist[0]

	awkstatement := fmt.Sprintf("{print $%d}'", awk)
	var b bytes.Buffer
	if err := Popen3(&b,
		exec.Command(arglist[0], arglist[1:]...),
		exec.Command("head", "-1"),
		exec.Command("awk", awkstatement),
	); err != nil {
		log.Fatalln(err)
	}
	return b.String(), nil
}

func Popen3GrepFast(cmd string, musthave string, mustnothave string) ([]string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd)
		fmt.Println("\tmusthave = ", musthave)
		fmt.Println("\tmustnothave = ", mustnothave)
		return []string{}, nil
	}

	var b bytes.Buffer
	arglist := strings.Split(cmd, " ")

	//check for len<2 and return null array

	var greplist []string
	if len(musthave) != 0 {
		greplist = strings.Split(musthave, "&")
	}
	var antigreplist []string
	if len(mustnothave) != 0 {
		antigreplist = strings.Split("-v "+mustnothave, "&")
	}
	//app:=arglist[0]
	var err error

	//case 1: popen and grep
	if len(greplist) > 0 && len(antigreplist) == 0 {
		if err = Popen3(&b,
			exec.Command(arglist[0], arglist[1:]...),
			exec.Command("grep", greplist[0:]...),
		); err != nil {
			log.Fatalln(err)
		}
	} else if len(greplist) == 0 && len(antigreplist) > 0 {

		//case 2: popen and antigrep
		if err = Popen3(&b,
			exec.Command(arglist[0], arglist[1:]...),
			exec.Command("grep", antigreplist[0:]...),
		); err != nil {
			log.Fatalln(err)
		}
	} else if len(greplist) > 0 && len(antigreplist) > 0 {
		//case 3: popen, grep and antigrep

		if err = Popen3(&b,
			exec.Command(arglist[0], arglist[1:]...),
			exec.Command("grep", greplist[0:]...),
			exec.Command("grep", antigreplist[0:]...),
		); err != nil {
			log.Fatalln(err)
		}
	}
	var bstring = b.String()
	var slice []string
	if len(strings.TrimSpace(bstring)) != 0 {
		slice = strings.Split(b.String(), "\n")
	}
	if len(slice) < 2 {
		return slice, nil
	}
	return slice[:len(slice)-1], nil
}

func Popen3Grep2(cmd string, musthave string, mustnothave string) ([]string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd)
		fmt.Println("\tmusthave = ", musthave)
		fmt.Println("\tmustnothave = ", mustnothave)
		return []string{}, nil
	}
	var b bytes.Buffer
	arglist := strings.Split(cmd, " ")

	var slice []string
	var greplist []string
	if len(musthave) != 0 {
		greplist = strings.Split(musthave, "&")
	}
	var antigreplist []string
	if len(mustnothave) != 0 {
		antigreplist = strings.Split(mustnothave, "&")
		for i := range antigreplist {
			antigreplist[i] = "-v " + antigreplist[i]
		}
	}

	var err error
	if err = Popen3(&b, exec.Command(arglist[0], arglist[1:]...)); err != nil {
		return slice, err
	}

	var bstring = b.String()
	if len(strings.TrimSpace(bstring)) != 0 {
		slice = strings.Split(b.String(), "\n")
	}

	newslice := make([]string, 0)
	for i := 0; i < len(slice); i++ {
		if len(strings.TrimSpace(slice[i])) != 0 {
			include := true
			for _, must := range greplist {
				if !strings.Contains(slice[i], must) {
					include = false
					break
				}
			}
			for _, mustnot := range antigreplist {
				if strings.Contains(slice[i], mustnot) {
					include = false
					break
				}
			}
			if include {
				newslice = append(newslice, slice[i])
			}
		}
	}

	return newslice, nil
}
func Popen3Grep(cmd string, musthave string, mustnothave string) ([]string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", cmd)
		fmt.Println("\tmusthave = ", musthave)
		fmt.Println("\tmustnothave = ", mustnothave)
		return []string{}, nil
	}
	var b bytes.Buffer
	arglist := strings.Split(cmd, " ")

	//check for len<2 and return null array
	var slice []string
	var greplist []string
	if len(musthave) != 0 {
		greplist = strings.Split(musthave, "&")
	}
	var antigreplist []string
	if len(mustnothave) != 0 {
		antigreplist = strings.Split(mustnothave, "&")
	}
	//app:=arglist[0]
	var err error

	if err = Popen3(&b, exec.Command(arglist[0], arglist[1:]...)); err != nil {
		return slice, err
	}

	var bstring = b.String()

	if len(strings.TrimSpace(bstring)) != 0 {
		slice = strings.Split(b.String(), "\n")
	}

	newslice := make([]string, 0)
	for i := 0; i < len(slice); i++ {
		//case 1: there are neither musthaves nor mustnothaves

		if len(strings.TrimSpace(slice[i])) != 0 {

			if len(greplist) == 0 && len(antigreplist) == 0 {
				//not really grep at all.. just expensive popen
				newslice = append(newslice, slice[i])
			} else if len(greplist) > 0 && len(antigreplist) == 0 {
				// only look at musthaves
				if SliceContains(greplist, slice[i]) {
					newslice = append(newslice, slice[i])
				}
			} else if len(musthave) == 0 && len(antigreplist) > 0 {
				if !SliceContains(antigreplist, slice[i]) {
					newslice = append(newslice, slice[i])
				}
			} else {
				if SliceContains(greplist, slice[i]) && !SliceContains(antigreplist, slice[i]) {
					newslice = append(newslice, slice[i])
				}
			}
		}
	}

	return newslice, nil
}

func SSHPopenToString(hostname string, command string) (string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", command)
		fmt.Println("\thostname = ", hostname)
		return "", nil
	}
	client, session, err := getsshclient(hostname)
	if err != nil {
		//panic(err)
		return "", err
	}
	out, err := session.CombinedOutput(command)
	if err != nil {
		//panic(err)
		return "", err
	}
	//fmt.Println(string(out))
	client.Close()
	return strings.TrimSpace(string(out)), nil
}

// write otuput content (including new lines) to file outputfile (including full absolute path) on remoteserver over SSH
func StringToFileOverSSH(outputContent string, remoteserver string, outputfile string) error {
	client, session, err := getsshclient(remoteserver)
	if err != nil {
		return err
	}
	defer client.Close()
	var stdin io.WriteCloser
	stdin, err = session.StdinPipe()
	if err != nil {
		log.Fatalf("Unable to setup stdin for session: %v", err)
	}
	go func() {
		defer stdin.Close()
		io.Copy(stdin, bytes.NewReader([]byte(outputContent+"\n")))
	}()

	if err = session.Run("cat >" + outputfile); err != nil {
		return err
	}
	return nil
}

func getsshclient(host string) (*ssh.Client, *ssh.Session, error) {

	key, err := ioutil.ReadFile(GetPrivateSSHKey())
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}
	sshConfig := &ssh.ClientConfig{
		User: os.Getenv("USER"),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	sshConfig.SetDefaults()
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func DmidecodeProduct() (string, error) {
	if GetDryRun() {
		fmt.Println("DRYRUN: ", "dmidecode")
		return "", nil
	}
	var b bytes.Buffer
	if err := Popen3(&b,
		exec.Command("dmidecode"),
		exec.Command("grep", "-i", "product"),
		exec.Command("head", "-1"),
		exec.Command("awk", "{print $3}"),
	); err != nil {
		log.Fatalln(err)
	}
	return b.String(), nil
}

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func MoveFile(src string, dst string) error {
	if err := RemoveFile(dst); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func GetFirstFile(rootpath string, hint string) string {

	var result string

	err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, hint) {
			//VerbosePrintln("found: " + path)
			result = path
		}

		return nil
	})

	if err != nil {
		fmt.Printf("walk error [%v]\n", err)
	}
	return result
}

func ExecToFile(cli string, ofile string) (err error) {
	if GetDryRun() {
		fmt.Printf("DRYRUN: %s > %s\n", cli, ofile)
		return nil
	}
	err = nil
	arglist := strings.Split(cli, " ")
	var outfile *os.File
	cmd := exec.Command(arglist[0], arglist[1:]...)

	// open the out file for writing
	if !strings.EqualFold(ofile, "") {
		outfile, err = os.Create(ofile)
		if err != nil {
			panic(err)
		}
		defer outfile.Close()
		cmd.Stdout = outfile
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	err = cmd.Wait()
	return
}

func Spinny(sigChan chan bool) {
	quit := false
	for !quit {

		s := "|/-\\"
		for i := 0; i < len(s); i++ {
			fmt.Printf("%c", s[i])
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b")
		}

		select {
		// case msg1 := <-messages:
		// 	//fmt.Println("received", msg1)
		// 	quit = true
		case sig := <-sigChan:
			quit = sig //fmt.Println("received signal", sig)
		default:
			//fmt.Println("not yet")
		}

	}
	fmt.Printf(" \b")
}

func System3AndSpin(cmd string, redirect string) (err error) {
	if GetDryRun() {
		fmt.Printf("DRYRUN: %s > %s\n", cmd, redirect)
		return nil
	}
	//var errorRec error
	var wg sync.WaitGroup
	wg.Add(1)
	//messages := make(chan string)
	sigChan := make(chan bool)
	errorChan := make(chan error)
	go func() {
		defer wg.Done()
		defer close(errorChan)
		defer close(sigChan)
		e := ExecToFile(cmd, redirect)
		sigChan <- true
		errorChan <- e
	}()
	go Spinny(sigChan)
	//errorRec = <-errorChan
	err = <-errorChan
	wg.Wait()
	return err
}

func Error2ExitCode(err error) int {
	//fmt.Printf("err was %q\n", err.Error())
	if strings.HasPrefix(err.Error(), "exit status ") {
		r, _ := strconv.ParseInt(err.Error()[12:], 10, 32)
		return int(r)
	}
	return 255
}

func Hostname() (string, error) {
	var hostname string
	var err error
	if hostname, err = PopentoString("hostname -s"); err != nil {
		return "localhost", err
	}

	return strings.TrimSpace(hostname), nil
}

func FileExistsEasy(p string) bool {
	b, err := FileExists(p)
	if err != nil {
		return false
	}
	return b
}

func RecursiveDelete(path string) error {
	if GetDryRun() {
		fmt.Println("DRYRUN: rm -rf " + path)
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if err := RecursiveDelete(entryPath); err != nil {
				return err
			}
		} else {
			VerbosePrintln("rm " + entryPath)
			if err := os.Remove(entryPath); err != nil {
				return err
			}
		}
	}
	VerbosePrintln("rmdir " + path)
	return os.Remove(path)
}

func RemoveGlob(glob string) error {
	if GetDryRun() {
		fmt.Println("DRYRUN: rm " + glob)
		return nil
	}
	if len(glob) == 0 || len(glob) == 1 {
		return errors.New("glob was empty or too short")
	}
	if !strings.HasSuffix(glob, "*") {
		VerbosePrintln("rm " + glob)
		return os.Remove(glob)
	}
	matchingFiles, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	if len(matchingFiles) == 0 {
		VerbosePrintln("no files matched pattern, returning")
		return nil
	}
	for _, file := range matchingFiles {
		VerbosePrintln("rm -f " + file)
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetEnvironment(v *bool, env_var string) {
	if *v || !strings.EqualFold(strings.TrimSpace(os.Getenv(env_var)), "") {

		if *v {

			VerbosePrintln("setting " + env_var + " true because of internal variable")
		}

		if !strings.EqualFold(strings.TrimSpace(os.Getenv(env_var)), "") {
			VerbosePrintln(fmt.Sprintf("elaborate env check thinks %s = %q is set", env_var, os.Getenv(env_var)))

		}

		VerbosePrintln("env var = " + env_var + " setting to true")

		*v = true
		os.Setenv(env_var, "1")
	} else {
		os.Unsetenv(env_var)
		*v = false
	}
}

func RemoveFile(path string) error {
	if !FileExistsEasy(path) {
		return nil
	}
	return os.Remove(path)
}

func MkdirAll(path string) error {
	if FileExistsEasy(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

type jpsProcStruct struct {
	pid       int
	className string
}

//USAGE: (local)
// jlist,err:=JPS() //list of all local java processes
// pidlist:=JPSStructListToIntList(SliceToJPSStruct(jlist),"BucketMigrator") // convert to list of pids
//
// or
//
// fmt.Println(GetStringListFromJPS(SliceToJPSStruct(jlist)))

//USAGE: (remote)
// jlist,err:=s.RemoteJPS()
// pidlist:=JPSStructListToIntList(SliceToJPSStruct(jlist),"BucketMigrator") // convert to list of pids
//
// or
//
// fmt.Println(GetStringListFromJPS(SliceToJPSStruct(jlist)))

func JPS() ([]string, error) {
	return Popen3Grep("ps aux", "java", "grep")
}

func SlicetoJPSStruct(lines []string, hint string) []jpsProcStruct {
	// Iterate through the lines and filter for Java processes
	var c string
	var result []jpsProcStruct
	for _, line := range lines {
		c = ""
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, err := strconv.Atoi(fields[1])
			if err != nil {
				panic(err.Error())
			}
			if len(hint) > 0 {
				for f := 0; f < len(fields); f++ {
					if strings.Contains(fields[f], hint) {
						c = GetClassNameLastSegment(fields[f])
					}
				}
			}
			if len(c) == 0 {
				if strings.Contains(fields[len(fields)-1], "/") && strings.Contains(fields[len(fields)-2], "/") {
					c = GetClassNameLastSegment(fields[len(fields)-3])
				} else if strings.EqualFold(strings.ToLower(fields[len(fields)-1]), "start") {
					c = GetClassNameLastSegment(fields[len(fields)-2])
				} else {
					c = GetClassNameLastSegment(fields[len(fields)-1])
				}
			}
			if len(c) == 0 {
				result = append(result, jpsProcStruct{pid: pid, className: "Unkown"})
			} else {
				result = append(result, jpsProcStruct{pid: pid, className: c})
			}
		}
	}
	return result
}

func JPSStructListToIntList(jps []jpsProcStruct, classname string) []int {
	var result []int
	for i := 0; i < len(jps); i++ {
		if strings.EqualFold(strings.ToLower(jps[i].className), strings.ToLower(classname)) {
			result = append(result, jps[i].pid)
		}
	}
	return result
}

func (j jpsProcStruct) ToString() string {
	return fmt.Sprintf("%d %s", j.pid, j.className)
}

func GetStringListFromJPS(jlist []jpsProcStruct) []string {
	var slist []string
	for i := 0; i < len(jlist); i++ {
		slist = append(slist, jlist[i].ToString())
	}
	return slist
}
func (j jpsProcStruct) GetPid() int {
	return j.pid
}
func (j jpsProcStruct) GetClassName() string {
	return j.className
}

func ParseThreadCount(status string) (int, error) {
	lines := strings.Split(status, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Threads:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strconv.Atoi(parts[1])
			}
		}
	}
	return 0, errors.New("thread Count not found")
}

func GetThreadCount(pid int) (int, error) {
	if pid == -1 {
		//if the process is dead, it's not using threads; so return 0
		return 0, nil
	}
	statusBytes, err := os.ReadFile(fmt.Sprintf(pid_status_format, pid))
	if err != nil {
		return 0, err
	}
	return ParseThreadCount(string(statusBytes))
}

const no_processes_found = "no processes found"

func CaptureJavaPid(jvm string, hint string) (int, error) {
	VerbosePrintln("BEGIN CapturePid(" + jvm + ")")
	lines, err := JPS()
	if err != nil {
		VerbosePrintf("\treturning with err: %s", err.Error())
		return 0, err
	}
	jlist := JPSStructListToIntList(SlicetoJPSStruct(lines, hint), jvm)
	if len(jlist) == 0 {
		return 0, fmt.Errorf(no_processes_found)
	}
	if len(jlist) > 1 {
		return jlist[0], fmt.Errorf("multiple processes found, returning first")
	}
	VerbosePrintln("END CapturePid(" + jvm + ")")
	return jlist[0], nil
}

func GetProcessList(mainHint string) ([]string, error) {
	var capture string
	err := System3toCapturedString(&capture, "ps aux")

	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(capture, "\n")
	var result []string
	for _, line := range lines {
		if strings.Contains(line, mainHint) {
			result = append(result, line)
		}
	}
	return result, nil
}

func CapturePids(main string, hint string) ([]int, error) {
	VerbosePrintf("BEGIN CapturePids(%s,%s)", main, hint)
	lines, err := GetProcessList(main)
	if err != nil {
		VerbosePrintf("\treturning with err: %s", err.Error())
		return []int{}, err
	}
	var pidList []int
	for _, line := range lines {
		VerbosePrintf("line=%s", line)
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, err := strconv.Atoi(fields[1])
			if err != nil {
				panic(err.Error())
			}
			pidList = append(pidList, pid)
		}
	}

	VerbosePrintf("END CapturePids(%s,%s)", main, hint)
	return pidList, nil
}

func GetArgsFromPid(pid int) ([]string, error) {
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf(pid_cmdline_format, pid))
	if err != nil {
		return nil, err
	}
	return strings.Split(string(cmdlineBytes), "\x00"), nil
}

func LoadCredFileMap(filename string) (map[string]string, error) {
	if !FileExistsEasy(filename) {
		return map[string]string{}, nil
	}
	credentials := make(map[string]string)

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into user and password
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}

		// Trim whitespace around the user and password
		user := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		// Store the user and password in the map
		credentials[user] = password
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return credentials, nil
}

func FileAuthenticatePasscode(passcode, filename string) bool {
	contents, err := ReadFileToSlice(filename, true)

	if err != nil {
		panic(err.Error())
	}

	for _, line := range contents {
		if strings.EqualFold(line, "passcode:"+passcode) {
			return true
		}
	}
	return false
}

func FileAuthenticate(username, password, filename string) bool {
	log.Printf("FileAuthenticate(%s,%s,%s)", username, password, filename)
	if !FileExistsEasy(filename) {
		return false
	}
	credmap, err := LoadCredFileMap(filename)

	if err != nil {
		panic(err.Error())
	}
	log.Printf("credmap=%v", credmap)
	log.Printf("credmap[%s]=%s", username, credmap[username])
	log.Printf("password=%s\n", password)
	log.Printf("strings.EqualFold(credmap[%s],%s)=%v\n", username, password, strings.EqualFold(credmap[username], password))
	return strings.EqualFold(credmap[username], password)
}

func IsCgoEnabled() bool {
	//	return !(runtime.GOOS != "js" && runtime.GOOS != "wasip1" && runtime.Compiler != "gccgo")

	buildInfo, _ := debug.ReadBuildInfo()

	for _, setting := range buildInfo.Settings {
		if setting.Key == "CGO_ENABLED" {
			if setting.Value == "1" {
				return true
			}
		}
	}

	return false
}

func GenerateFilename(f string, suffix string) FilenameStruct {
	var fns FilenameStruct
	hasStats := true
	fns.Parse(f)
	if err := fns.GetStat(); err != nil {
		hasStats=false
	}
	newFileBase := fns.GetBase() + suffix
	if hasStats && !fns.hasDate {
		newFileBase += fns.GetModTime()
	}
	fns.SetBase(newFileBase)
	fns.SetFullName(fmt.Sprintf("%s/%s%s", fns.GetPath(), fns.GetBase(), fns.GetExt()))
	return fns
}

func GenerateMoveCLI(f string, suffix string) string {
	fns := GenerateFilename(f, suffix)
	return fmt.Sprintf("mv %s %s", f, fns.GetFullName())
}

func IsMounted(mntpt string) bool {
	if !FileExistsEasy("/proc") {
		panic("runtime error: OS does not support proc")
	}
	haystack, err := ReadFileToSlice("/proc/mounts", false)

	if err != nil {
		panic("runtime error: unable to read from /proc/mounts")
	}

	// SliceContains(slice, mntpt)
	for _, h := range haystack {
		if strings.Contains(h, mntpt) {
			return true
		}

	}
	return false
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
