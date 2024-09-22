// Copyright 2023 Chris Delezenski <chris.delezenski@gmail.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alfredo

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// the only global variable, Verbose is only used directly by the VerbosePrintln() function
var verbose bool
var force bool
var debug bool
var panicOnFail bool
var experimental bool

func SetVerbose(v bool) {
	verbose = v
	os.Setenv("VERBOSE", "1")
}
func GetVerbose() bool {
	return getState("VERBOSE", verbose) || getState("DEBUG", debug)
}

func SetDebug(b bool) {
	debug = b
	os.Setenv("DEBUG", "1")
}

func GetDebug() bool {
	return getState("DEBUG", debug)
}
func SetForce(f bool) {
	force = f
	os.Setenv("FORCE", "1")
}

func GetForce() bool {
	return getState("FORCE", force)
}

func SetPanic(p bool) {
	panicOnFail = p
	os.Setenv("PANIC", "1")
}
func GetPanic() bool {
	return getState("PANIC", panicOnFail)
}

func getState(e string, s bool) bool {
	return strings.EqualFold(os.Getenv(e), "1") || s
}

func GetExperimental() bool {
	return getState("EXPERIMENTAL", experimental)
}

func SetExperimental(e bool) {
	experimental = e
	os.Setenv("EXPERIMENTAL", "1")
}

func PanicError(msg string) error {
	if GetPanic() {
		panic(msg)
	}
	return errors.New(msg)
}

// for debugging purposes, a more verbose output to catch attention
func VerbosePrintln(cmd string) {
	if verbose {
		fmt.Println("::::", cmd, "::::")
	}
}

func VerbosePrintf(format string, a ...any) {
	VerbosePrintln(fmt.Sprintf(format, a...))
}

func DebugPrintln(cmd string) {
	if verbose {
		fmt.Println("#:DEBUG:", cmd, "::::")
	}
}

func DebugPrintf(format string, a ...any) {
	DebugPrintln(fmt.Sprintf(format, a...))
}

func CommentPrintln(cmd string) {
	fmt.Println("# " + cmd)
}

func CommentPrintf(format string, a ...any) {
	CommentPrintln(fmt.Sprintf(format, a...))
}

// load a file with lines that \n terminated into a slice (used by the syslog self-test)
func LoadFileIntoSlice(f string) ([]string, error) {
	var returnlist []string
	if !FileExistsEasy(f) {
		return nil, errors.New("File " + f + " does not exist")
	}
	// open the file using Open() function from os library
	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// read the file line by line using a scanner
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		returnlist = append(returnlist, scanner.Text())
	}
	// check for the error that occurred during the scanning
	return returnlist, scanner.Err()
}

func EatError(s string, e error) string {
	if e != nil {
		panic(e.Error())
	}
	return s
}

func YoungestFileTime() string {
	var youngestTime time.Time
	//var youngestFile string
	var count int
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			modTime := info.ModTime()
			if modTime.After(youngestTime) {
				youngestTime = modTime
				fmt.Println("young file found=" + path)
				//youngestFile = path
			}
			count++
		}
		return nil
	})

	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Processed %d files.\n", count)
	timesplit := strings.Split(youngestTime.String(), " ")
	fmt.Printf("time: %s\n", youngestTime.String())
	parsedDate, err := time.Parse("2006-01-02", timesplit[0])
	if err != nil {
		panic(err.Error())
	}

	// Format the date in the desired output format
	return parsedDate.Format("02Jan2006")
}

type FilenameStruct struct {
	fullname string
	base     string
	path     string
	ext      string
	hasDate  bool
	modtime  string
}

func (fns FilenameStruct) GetFullName() string {
	return fns.fullname
}
func (fns *FilenameStruct) SetFullName(f string) {
	fns.fullname = f
}
func (fns FilenameStruct) WithFullName(f string) FilenameStruct {
	fns.fullname = f
	return fns
}

func (fns FilenameStruct) GetExt() string {
	return fns.ext
}
func (fns *FilenameStruct) SetExt(e string) {
	fns.ext = e
}
func (fns FilenameStruct) WithExt(e string) FilenameStruct {
	fns.ext = e
	return fns
}
func (fns FilenameStruct) GetModTime() string {
	return fns.modtime
}
func (fns *FilenameStruct) SetModTime(mt string) {
	fns.modtime = mt
}
func (fns FilenameStruct) WithModTime(mt string) FilenameStruct {
	fns.modtime = mt
	return fns
}
func (fns FilenameStruct) GetBase() string {
	return fns.base
}
func (fns *FilenameStruct) SetBase(b string) {
	fns.base = b
}
func (fns FilenameStruct) WithBase(b string) FilenameStruct {
	fns.base = b
	return fns
}

func (fns FilenameStruct) GetHasDate() bool {
	return fns.hasDate
}
func (fns *FilenameStruct) SetHasDate(b bool) {
	fns.hasDate = b
}
func (fns FilenameStruct) WithHasDate(b bool) FilenameStruct {
	fns.hasDate = b
	return fns
}

func (fns FilenameStruct) GetPath() string {
	return fns.path
}
func (fns *FilenameStruct) SetPath(p string) {
	fns.path = p
}
func (fns FilenameStruct) WithPath(p string) FilenameStruct {
	fns.path = p
	return fns
}

// FileBaseContainsDate : f contains a date
func FileBaseContainsDate(f string) bool {

	if len(f) < 7 {
		return false // shortcut, if base is less than 7, there's no date to look for
	}
	const max = 2025
	const min = 1995
	var datelist [max - min]int

	for i := 0; i < len(datelist); i++ {
		datelist[i] = i + min
	}

	for i := 0; i < len(datelist); i++ {
		for j := 1; j < 13; j++ {
			s := time.Month(j).String()
			lhs := f[len(f)-7:]
			rhs := s[0:3] + strconv.Itoa(datelist[i])
			if lhs == rhs {
				DebugPrintf("# found match - returning true: %s", lhs)
				return true
			}
		}
	}

	fmt.Println("# exhausted all options, returning false")
	return false
}

func (fns *FilenameStruct) GetStat() error {
	fileStat, err := os.Stat(fns.GetFullName())
	if err != nil {
		//wg.Done()
		return err
	}
	fns.modtime = fileStat.ModTime().Format("02Jan2006")
	return nil
}

func (fns *FilenameStruct) Parse(f string) error {
	if len(f) == 0 {
		return PanicError("Filename::Parse(): empty string passed in")
	}

	fns.fullname = f

	fns.ext = filepath.Ext(fns.fullname)

	l := len(fns.fullname)

	//tar.gz and tar.bz2 exception
	if l > 7 && fns.ext == ".gz" && fns.fullname[l-7:l] == ".tar.gz" {
		fns.ext = ".tar.gz"
	} else if l > 8 && fns.ext == ".bz2" && fns.fullname[l-8:l] == ".tar.bz2" {
		fns.ext = ".tar.bz2"
	} else if l > 8 && fns.ext == ".enc" && fns.fullname[l-8:l] == ".tgz.enc" {
		fns.ext = ".tgz.enc"
	}
	fns.base = filepath.Base(fns.fullname)[0 : len(filepath.Base(fns.fullname))-len(fns.ext)]

	fns.path = filepath.Dir(fns.fullname)

	fns.hasDate = FileBaseContainsDate(fns.base)

	//debugPrint("=====>filename " + f + " has date: " + strconv.FormatBool(fns.hasDate))

	VerbosePrintf("alfredo::util.go::FilenameStruct::Parse(%s)=%s", f, fns.GetFullName())

	return nil
}

func TestEndpoint(url string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func CompareMaps(mapA, mapB map[string]string) []string {
	var diff []string
	// Iterate over mapA and compare with mapB
	for key, valueA := range mapA {
		if valueB, found := mapB[key]; found {
			if valueA != valueB {
				diff = append(diff, fmt.Sprintf("Difference at key '%s': A=%v, B=%v", key, valueA, valueB))
			}
		} else {
			diff = append(diff, fmt.Sprintf("Key '%s' found in A but not in B", key))
		}
	}

	// Check for keys in mapB that are not in mapA
	for key := range mapB {
		if _, found := mapA[key]; !found {
			diff = append(diff, fmt.Sprintf("Key '%s' found in B but not in A", key))
		}
	}
	return diff
}
