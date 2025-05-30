package alfredo

import (
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestBoolMapContainer(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]bool
		itemsCSV string
		s        bool
		expected map[string]bool
	}{
		{
			name:     "Add items to map",
			initial:  map[string]bool{"item1": false},
			itemsCSV: "item1,item2,item3",
			s:        true,
			expected: map[string]bool{"item1": true, "item2": true, "item3": true},
		},
		{
			name:     "Delete items from map",
			initial:  map[string]bool{"item1": true, "item2": true, "item3": true},
			itemsCSV: "item2,item3",
			s:        false,
			expected: map[string]bool{"item1": true},
		},
		{
			name:     "Empty CSV string",
			initial:  map[string]bool{"item1": true},
			itemsCSV: "",
			s:        true,
			expected: map[string]bool{"item1": true},
		},
		{
			name:     "Empty records in CSV",
			initial:  map[string]bool{"item1": true},
			itemsCSV: ",,,",
			s:        true,
			expected: map[string]bool{"item1": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmc := &BoolMapContainer{bmc: tt.initial}
			bmc.FromCSV(tt.itemsCSV, tt.s)

			if !reflect.DeepEqual(bmc.bmc, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, bmc.bmc)
			}
		})
	}
}

func TestBoolMapContainer_ToSlice(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]bool
		expected []string
	}{
		{
			name:     "All items enabled",
			initial:  map[string]bool{"item1": true, "item2": true, "item3": true},
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "Some items enabled",
			initial:  map[string]bool{"item1": true, "item2": false, "item3": true},
			expected: []string{"item1", "item3"},
		},
		{
			name:     "No items enabled",
			initial:  map[string]bool{"item1": false, "item2": false, "item3": false},
			expected: []string{},
		},
		{
			name:     "Empty map",
			initial:  map[string]bool{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmc := BoolMapContainer{bmc: tt.initial}
			result := bmc.ToSlice()
			// Sorting to ensure order does not affect the comparison
			sort.Strings(result)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(result, tt.expected) && result != nil {
				t.Errorf("expected %d, got %d", len(tt.expected), len(result))
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTail(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5
Line 6
at com.example.Main.method(Main.java:42)
Line 7
Line 8
at java.util.ArrayList.get(ArrayList.java:500)
Line 9
Line 10`

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to the file
	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tmpFile.Close()

	// Test Tail function
	expected := []string{
		"Line 5",
		"Line 6",
		"Line 7",
		"Line 8",
		"Line 9",
		"Line 10",
	}
	result, err := Tail(tmpFile.Name(), 6)
	if err != nil {
		t.Fatalf("Tail failed: %v", err)
	}

	if !equalSlices(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestSplitLines(t *testing.T) {
	input := []byte("Line 1\nLine 2\nLine 3\n")
	expected := []string{"Line 1", "Line 2", "Line 3"}

	result := splitLines(input)
	if !equalSlices(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestFilterLines(t *testing.T) {
	input := []string{
		"Line 1",
		"at com.example.Main.method(Main.java:42)",
		"Line 2",
		"at java.util.ArrayList.get(ArrayList.java:500)",
		"Line 3",
	}
	expected := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	result := filterLines(input)
	if !equalSlices(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
func TestStringContainsInCSV(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		csv      string
		expected bool
	}{
		{
			name:     "String contains item in CSV",
			s:        "hello world",
			csv:      "hello,world",
			expected: true,
		},
		{
			name:     "String does not contain item in CSV",
			s:        "goodbye woorld",
			csv:      "hello,world",
			expected: false,
		},
		{
			name:     "Empty CSV string",
			s:        "hello world",
			csv:      "",
			expected: false,
		},
		{
			name:     "Empty input string",
			s:        "",
			csv:      "hello,world",
			expected: false,
		},
		{
			name:     "CSV contains empty item",
			s:        "hello world",
			csv:      "hello,,world",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringContainsInCSV(tt.s, tt.csv)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
func TestSliceToDoc(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "Single element slice",
			input:    []string{"apple"},
			expected: "apple",
		},
		{
			name:     "Two elements slice",
			input:    []string{"apple", "banana"},
			expected: "apple and banana",
		},
		{
			name:     "Multiple elements slice",
			input:    []string{"apple", "banana", "cherry"},
			expected: "apple, banana and cherry",
		},
		{
			name:     "Multiple elements slice with more items",
			input:    []string{"apple", "banana", "cherry", "date"},
			expected: "apple, banana, cherry and date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceToDoc(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRemoveKeyPartialFromSlice(t *testing.T) {
	type args struct {
		s     string
		slice []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Remove partially matching key",
			args: args{
				s:     "ban",
				slice: []string{"apple", "banana", "cherry"},
			},
			want: []string{"apple", "cherry"},
		},
		{
			name: "No partial match found",
			args: args{
				s:     "date",
				slice: []string{"apple", "banana", "cherry"},
			},
			want: []string{"apple", "banana", "cherry"},
		},
		{
			name: "Remove partial match from empty slice",
			args: args{
				s:     "apple",
				slice: []string{},
			},
			want: []string{},
		},
		{
			name: "Remove partial match with case insensitivity",
			args: args{
				s:     "BAN",
				slice: []string{"apple", "banana", "cherry"},
			},
			want: []string{"apple", "cherry"},
		},
		{
			name: "Remove key when multiple partial matches exist",
			args: args{
				s:     "ban",
				slice: []string{"apple", "banana", "band", "cherry"},
			},
			want: []string{"apple", "cherry"},
		},
		{
			name: "Remove partial match from single-element slice",
			args: args{
				s:     "app",
				slice: []string{"apple"},
			},
			want: []string{},
		},
		{
			name: "Remove partial match that matches the last element",
			args: args{
				s:     "cher",
				slice: []string{"apple", "banana", "cherry"},
			},
			want: []string{"apple", "banana"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveKeyPartialFromSlice(tt.args.s, tt.args.slice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveKeyPartialFromSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
