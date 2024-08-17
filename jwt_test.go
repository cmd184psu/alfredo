// Copyright 2024 C Delezenski <cmd184psu@gmail.com>
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
	"encoding/base64"
	"testing"
)

func TestGenerateJWTKey(t *testing.T) {
	// Generate the key
	key := GenerateJWTKey()

	// Test the length of the generated key
	expectedLength := 43 // 32 bytes -> base64 encoded length is 43 characters
	if len(key) != expectedLength {
		t.Errorf("GenerateJWTKey() length = %d; expected %d", len(key), expectedLength)
	}

	// Test if the key is a valid base64 URL encoding
	_, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		t.Errorf("GenerateJWTKey() returned an invalid base64 URL encoding: %v", err)
	}
}
