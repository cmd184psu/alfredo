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
	"os"
	"path/filepath"
	"strings"
	"time"
)

// the only global variable, Verbose is only used directly by the VerbosePrintln() function
var verbose bool

func SetVerbose(v bool) {
	verbose = v
	os.Setenv("VERBOSE", "1")
}
func GetVerbose() bool {
	return strings.EqualFold(os.Getenv("VERBOSE"), "1") || verbose
}

// for debugging purposes, a more verbose output to catch attention
func VerbosePrintln(cmd string) {
	if verbose {
		fmt.Println("::::", cmd, "::::")
	}
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
