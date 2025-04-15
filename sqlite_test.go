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

func TestDatabaseStruct_Count(t *testing.T) {
	type fields struct {
		DbPath string
		Table  string
		exe    *CLIExecutor
	}
	type args struct {
		sel   string
		where string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantRes int
	}{
		{
			name: "Count elements with status > 10",
			fields: fields{
				DbPath: "database.db",
				Table:  "t_s_mpu_stream",
			},
			args: args{
				sel:   "key",
				where: "lastModified > 1744685619000",
			},
			wantErr: false,
			wantRes: 0, // Replace with the expected count from your database
		},
		{
			name: "Count with invalid column",
			fields: fields{
				DbPath: "database.db",
				Table:  "t_s_mpu_stream",
				exe:    NewCLIExecutor(),
			},
			args: args{
				sel:   "invalid_column",
				where: "status > 10",
			},
			wantErr: true,
			wantRes: 0,
		},
		{
			name: "Count with empty where clause",
			fields: fields{
				DbPath: "database.db",
				Table:  "t_s_mpu_stream",
				exe:    NewCLIExecutor(),
			},
			args: args{
				sel:   "key",
				where: "",
			},
			wantErr: false,
			wantRes: 11, // Replace with the total count of elements in your table
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewSQLiteDB().WithDbPath(tt.fields.DbPath).WithTable(tt.fields.Table)

			err := db.Count(tt.args.sel, tt.args.where)

			// fmt.Printf("echo %q | %s\n", db.exe.GetRequestPayload(), db.exe.GetCli())
			// fmt.Println("exit code: ", db.exe.GetStatusCode())
			// fmt.Println("output: ", db.exe.GetResponseBody())

			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseStruct.Count() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				gotRes := db.GetResultInt()
				if gotRes != tt.wantRes {
					t.Errorf("DatabaseStruct.Count() result = %v, want %v", gotRes, tt.wantRes)
				}
			}
		})
	}
}
