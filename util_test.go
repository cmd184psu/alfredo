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

import "testing"

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
