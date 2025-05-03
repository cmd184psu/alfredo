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
	"path/filepath"
)

// simply write a data interface to a json file; shamelessly picked off the internet
func WriteStructToJSONFile(filePath string, structure interface{}) error {
	if !FileExistsEasy(filepath.Dir(filePath)) {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	}
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

// simply write a data interface to a json file; shamelessly picked off the internet
func WriteStructToJSONFilePP(filePath string, structure interface{}) error {
	// file, err := os.Create(filePath)
	// if err != nil {
	// 	return err
	// }
	// defer file.Close()

	// s := PrettyPrint(structure)
	// if len(s) == 0 {
	// 	return errors.New("string was empty on pp")
	// } else {
	// 	fmt.Printf("size is %d\n", len(s))
	// 	fmt.Println(s)
	// }
	return WriteStringToFile(filePath, PrettyPrint(structure))
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
	if err := System3toCapturedString(&s, cli); err != nil {
		return err
	}
	// VerbosePrintln("Alfredo:::ReadStructFromCommand(" + cli + ")")
	// VerbosePrintln("\ts=" + s)
	return ReadStructFromString(s, &structure)
}

func ReadStructFromCommandOverSSH(ssh SSHStruct, cli string, structure interface{}) error {
	var err error
	ssh.capture = true
	err = ssh.SecureRemoteExecution(cli)
	if err != nil {
		return err
	}
	return ReadStructFromString(ssh.stdout, &structure)
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

func PrettyPrintJSONFile(filePath string) {
	// Replace "your_file.json" with the actual path to your JSON file

	// Read the JSON file into memory
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	// Create a map to store the JSON data
	var data map[string]interface{}

	// Unmarshal the JSON data into the map
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
		return
	}

	// Pretty print the JSON data
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error pretty printing JSON data:", err)
		return
	}

	// Display the pretty-printed JSON on stdout
	fmt.Println(string(prettyJSON))
}

func TranslateSimilarStructure(input interface{}, output *interface{}) error {
	// Marshal the input structure into a []byte
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("error marshaling: %v", err)
	}

	// Unmarshal the []byte into the output structure
	err = json.Unmarshal(data, output)
	if err != nil {
		return fmt.Errorf("error unmarshaling: %v", err)
	}

	return nil
}

func JsonDeepCopy(input interface{}) (interface{}, error) {
	// Marshal the input structure into a []byte
	data, err := json.Marshal(input)
	if err != nil {
		fmt.Printf("error marshaling: %v\n", err)
		return nil, err
	}

	// Unmarshal the []byte into the output structure
	var output interface{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		fmt.Printf("error unmarshaling: %v\n", err)
		return nil, err
	}

	return output, nil
}
