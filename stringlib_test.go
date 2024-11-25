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
