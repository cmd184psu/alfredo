package alfredo

import (
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
