// Copyright 2023 Chris Delezensk <chris.delezenski@gmail.com>
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
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// simply write a data interface to a json file; shamelessly picked off the internet
func WriteStructToJSONFile(filePath string, structure interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(structure)
	if err != nil {
		return err
	}
	return file.Chmod(0644)
}

// simply reads JSON data from a file and populates the provided structure.
func ReadStructFromJSONFile(filePath string, structure interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, structure)
}
func ReadStructFromString(content string, structure interface{}) error {
	return json.Unmarshal([]byte(content), structure)
}

func ReadStructFromCommand(cli string, structure interface{}) error {
	var s string
	var err error
	if s, err = PopentoString(cli); err != nil {
		panic(err.Error())
	}
	return ReadStructFromString(s, &structure)
}

func ReadStructFromCommandOverSSH(ssh SSHStruct, cli string, structure interface{}) error {
	var err error
	ssh.capture = true
	ssh, err = ssh.SecureRemoteExecution(cli)
	if err != nil {
		return err
	}
	return ReadStructFromString(ssh.body, &structure)
}

// generate a "prettyprint" output of the structure
func PrettyPrint(v any) string {
	jsonData, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		fmt.Println("Error marshalling JSON: " + err.Error())
		return "{}"
	}
	return string(jsonData)
}