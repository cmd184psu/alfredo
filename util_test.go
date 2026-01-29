// Copyright 2024 C Delezenski <chris.delezenski@gmail.com>
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
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestFileBaseContainsDate(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// Test cases
		{"file_Jan2023", true}, // valid date
		{"file_Feb2010", true}, // valid date
		{"file_Mar1995", true}, // valid date, edge case at lower bound
		//		{"file_Dec2025", true},  // valid date, edge case at upper bound
		{"file_Jan1994", false}, // invalid date, before min year
		{"file_Jan2026", false}, // invalid date, after max year
		{"file_abc2023", false}, // invalid month abbreviation
		{"file_Jan20", false},   // insufficient length
		{"file_2023", false},    // no month, just the year
		//		{"Jan2023_file", true},  // valid date at end of string
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := FileBaseContainsDate(tt.filename)
			if result != tt.expected {
				t.Errorf("FileBaseContainsDate(%s) = %v; expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestCompareMaps(t *testing.T) {
	tests := []struct {
		name     string
		mapA     map[string]string
		mapB     map[string]string
		expected []string
	}{
		{
			name: "Identical maps (1)",
			mapA: map[string]string{"g:slegacyg1;u:slegacyu1,ak:00a9876ea429fd097ac5": "hJMULP9z/hIsTv4+HbFAXCey59j0fpheGZ+6aF2R",
				"g:slegacyg2;u:slegacyu2,ak:0378ef942fb1b3baaf19": "FcaonXUuzHYSIyYMl88gkwTYbNN611vp5BrI+xjm"},
			mapB: map[string]string{"g:slegacyg1;u:slegacyu1,ak:00a9876ea429fd097ac5": "hJMULP9z/hIsTv4+HbFAXCey59j0fpheGZ+6aF2R",
				"g:slegacyg2;u:slegacyu2,ak:0378ef942fb1b3baaf19": "FcaonXUuzHYSIyYMl88gkwTYbNN611vp5BrI+xjm"},

			expected: []string{},
		},
		{
			name:     "Identical maps",
			mapA:     map[string]string{"a": "1", "b": "2"},
			mapB:     map[string]string{"a": "1", "b": "2"},
			expected: []string{},
		},
		{
			name:     "Different values",
			mapA:     map[string]string{"a": "1", "b": "2"},
			mapB:     map[string]string{"a": "1", "b": "3"},
			expected: []string{"Difference at key 'b': A=2, B=3"},
		},
		{
			name:     "Key in A not in B",
			mapA:     map[string]string{"a": "1", "b": "2", "c": "3"},
			mapB:     map[string]string{"a": "1", "b": "2"},
			expected: []string{"Key 'c' found in A but not in B"},
		},
		{
			name:     "Key in B not in A",
			mapA:     map[string]string{"a": "1", "b": "2"},
			mapB:     map[string]string{"a": "1", "b": "2", "c": "3"},
			expected: []string{"Key 'c' found in B but not in A"},
		},
		{
			name: "Multiple differences",
			mapA: map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"},
			mapB: map[string]string{"a": "1", "b": "3", "d": "5", "e": "6"},
			expected: []string{
				"Difference at key 'b': A=2, B=3",
				"Key 'c' found in A but not in B",
				"Difference at key 'd': A=4, B=5",
				"Key 'e' found in B but not in A",
			},
		},
		{
			name:     "Empty maps",
			mapA:     map[string]string{},
			mapB:     map[string]string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareMaps(tt.mapA, tt.mapB)

			// Sort the results and expected slices for consistent comparison
			sort.Strings(result)
			sort.Strings(tt.expected)

			if len(result) != 0 || len(tt.expected) != 0 {
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("CompareMaps() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func Test_getPassCodeFromSlice(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Valid passcode",
			args: args{s: []string{"# Comment", "passcode: 12345"}},
			want: "12345",
		},
		{
			name: "Empty slice",
			args: args{s: []string{}},
			want: "",
		},
		{
			name: "No passcode",
			args: args{s: []string{"# Comment", "another: value"}},
			want: "",
		},
		{
			name: "Passcode with spaces",
			args: args{s: []string{"# Comment", "passcode:  67890"}},
			want: "67890",
		},
		{
			name: "Passcode with leading spaces",
			args: args{s: []string{"# Comment", "  passcode: 54321"}},
			want: "54321",
		},
		{
			name: "Passcode with trailing spaces",
			args: args{s: []string{"# Comment", "passcode: 98765 "}},
			want: "98765",
		},
		{
			name: "Multiple passcodes",
			args: args{s: []string{"# Comment", "passcode: 11111", "passcode: 22222"}},
			want: "11111",
		},
		{
			name: "Passcode with special characters",
			args: args{s: []string{"# Comment", "passcode: @#$%^&*"}},
			want: "@#$%^&*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPassCodeFromSlice(tt.args.s); got != tt.want {
				t.Errorf("getPassCodeFromSlice(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
func TestGetNewFilenameWithDateStampEasy(t *testing.T) {

	exe:=NewCLIExecutor().WithDirectory("/tmp")

	touchList:=[]string{}
	touchList=append(touchList,"/tmp/example.txt")
	touchList=append(touchList,"/tmp/example")
	touchList=append(touchList,"/tmp/example.file.txt")
	touchList=append(touchList,"/tmp/example@123!.txt")

	for _, file:=range touchList{
		exe.WithCommand("touch " + file)
		if err:=exe.Execute();err!=nil{
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		defer func(f string) {
			exe2:=NewCLIExecutor()
			exe2.WithCommand("rm " + f)
			_ = exe2.Execute()
		}(file)
	}


	tests := []struct {
		name             string
		originalFilename string
		want             string
	}{
		{
			name:             "Valid filename with extension",
			originalFilename: "/tmp/example.txt",
			want:             "/tmp/example-02Jan2006.txt", // Replace "02Jan2006" with the actual date format during runtime
		},
		{
			name:             "Filename without extension",
			originalFilename: "/tmp/example",
			want:             "/tmp/example-02Jan2006", // Replace "02Jan2006" with the actual date format during runtime
		},
		{
			name:             "Filename with multiple dots",
			originalFilename: "/tmp/example.file.txt",
			want:             "/tmp/example.file-02Jan2006.txt", // Replace "02Jan2006" with the actual date format during runtime
		},
		{
			name:             "Filename with special characters",
			originalFilename: "/tmp/example@123!.txt",
			want:             "/tmp/example@123!-02Jan2006.txt", // Replace "02Jan2006" with the actual date format during runtime
		},
		{
			name:             "Empty filename",
			originalFilename: "",
			want:             "", // Expecting a panic or error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.originalFilename == "" {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for empty filename, but function did not panic")
					}
				}()
			}

			got := GetNewFilenameWithDateStampEasy(tt.originalFilename)

			// Replace "02Jan2006" in the expected result with the actual date during runtime
			expectedDate := time.Now().Format("02Jan2006")
			expected := strings.Replace(tt.want, "02Jan2006", expectedDate, 1)

			if got != expected {
				t.Errorf("GetNewFilenameWithDateStampEasy(%q) = %v, want %v", tt.originalFilename, got, expected)
			}
		})
	}
}
