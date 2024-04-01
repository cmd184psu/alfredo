package alfredo

import (
	"os"
	"reflect"
	"testing"
)

func TestGetFileFindCLI(t *testing.T) {
	type args struct {
		directoryPath string
		prefix        string
		glob          string
	}
	tests := []struct {
		name string
		args args
		want string
	}{

		{
			name: "find cli",
			args: args{
				directoryPath: "./",
				prefix:        "stringlib",
				glob:          "*.go",
			},
			want: "find . -iname \"stringlib*.go\"",
		},
		{
			name: "find cli 2",
			args: args{
				directoryPath: ".",
				prefix:        "stringlib",
				glob:          "*.go",
			},
			want: "find . -iname \"stringlib*.go\"",
		}, // TODO: Add test cases.
		{
			name: "find cli 3",
			args: args{
				directoryPath: ".",
				prefix:        "",
				glob:          "*.json",
			},
			want: "find . -iname \"*.json\"",
		}, // TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFileFindCLI(tt.args.directoryPath, tt.args.prefix, tt.args.glob); got != tt.want {
				t.Errorf("GetFileFindCLI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindFiles(t *testing.T) {
	dir, _ := os.Getwd()

	type args struct {
		directoryPath string
		prefix        string
		glob          string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "simple find",
			args: args{
				directoryPath: "./",
				prefix:        "stringlib",
				glob:          "*.go",
			},
			want: []string{"./stringlib.go",
				"./stringlib_test.go"},
		},
		{
			name: "simple find 2",
			args: args{
				directoryPath: dir,
				prefix:        "stringlib",
				glob:          "*.go",
			},
			want: []string{dir + "/stringlib.go",
				dir + "/stringlib_test.go"},
		},
		// {
		// 	name: "simple find (err)",
		// 	args: args{
		// 		directoryPath: "~/",
		// 		prefix:        "stringlib",
		// 		glob:          "*.go",
		// 	},
		// 	want: []string{"./stringlib.go",
		// 		"./junk/strtools/stringlib.go",
		// 		"./stringlib_test.go"},
		// 	wantErr: true,
		// },
		// {
		// 	name: "simple find",
		// 	args: args{
		// 		directoryPath: ".",
		// 		prefix:        "stringlib",
		// 		glob:          "*.go",
		// 	},
		// 	want: []string{"./stringlib.go",
		// 		"./junk/strtools/stringlib.go",
		// 		"./stringlib_test.go"},
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindFiles(tt.args.directoryPath, tt.args.prefix, tt.args.glob)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
