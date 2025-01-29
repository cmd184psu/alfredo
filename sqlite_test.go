package alfredo

import (
	"strings"
	"testing"
)

func TestDatabaseStruct_Query(t *testing.T) {
	db := NewSQLiteDB()
	db.LoadConfig("dbtest.json")

	// Call the Query method

	if err := db.Count("key", ""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	result := db.GetResult()
	if result != "101" {
		t.Errorf("expected result %q, got %q", "101", result)
	}
	if err := db.Sum("size", ""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	result = HumanReadableStorageCapacity(db.GetResultInt64())
	if !strings.EqualFold(result, "125.65 MiB") {
		t.Errorf("expected result %q, got %q", "125.65 MiB", result)
	}
}
