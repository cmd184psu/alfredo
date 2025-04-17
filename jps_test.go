package alfredo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func Test_parseClassName(t *testing.T) {
	type args struct {
		cmdLine string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseClassName(tt.args.cmdLine); got != tt.want {
				t.Errorf("parseClassName() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestGetJavaProcessesFromBytes(t *testing.T) {
	type args struct {
		output          []byte
		filterClassName string
	}
	type testStruct struct {
		name    string
		args    args
		want    []ProcessInfo
		wantErr bool
	}
	tests := []testStruct{}

	tests = append(tests, testStruct{

		name: "No processes",
		args: args{
			output:          []byte(""),
			filterClassName: "",
		},
		want:    []ProcessInfo{},
		wantErr: false,
	})

	everything, _ := ReadFileToSlice("pssample.txt", true)

	tests = append(tests, testStruct{

		name: "load everything from file, all at once, filter out special class",
		args: args{
			output:          []byte(strings.Join(everything, "\n")),
			filterClassName: "BucketMigration",
		},
		want: []ProcessInfo{
			{PID: 65056, ClassName: "BucketMigration"},
		},
		wantErr: false,
	})

	// {
	// 	name: "Single Java process without filter",
	// 	args: args{
	// 		output:          []byte("1234 java -jar myapp.jar\n"),
	// 		filterClassName: "",
	// 	},
	// 	want: []ProcessInfo{
	// 		{PID: 1234, ClassName: "myapp.jar"},
	// 	},
	// 	wantErr: false,
	// },
	// {
	// 	name: "Multiple Java processes without filter",
	// 	args: args{
	// 		output: []byte("1234 java -jar myapp.jar\n5678 java -cp . com.example.MainClass\n"),
	// 		filterClassName: "",
	// 	},
	// 	want: []ProcessInfo{
	// 		{PID: 1234, ClassName: "myapp.jar"},
	// 		{PID: 5678, ClassName: "com.example.MainClass"},
	// 	},
	// 	wantErr: false,
	// },
	// {
	// 	name: "Filter by className",
	// 	args: args{
	// 		output: []byte("1234 java -jar myapp.jar\n5678 java -cp . com.example.MainClass\n"),
	// 		filterClassName: "MainClass",
	// 	},
	// 	want: []ProcessInfo{
	// 		{PID: 5678, ClassName: "com.example.MainClass"},
	// 	},
	// 	wantErr: false,
	// },
	// {
	// 	name: "Invalid process line",
	// 	args: args{
	// 		output: []byte("invalid line\n1234 java -jar myapp.jar\n"),
	// 		filterClassName: "",
	// 	},
	// 	want: []ProcessInfo{
	// 		{PID: 1234, ClassName: "myapp.jar"},
	// 	},
	// 	wantErr: false,
	// },
	// {
	// 	name: "Non-Java process",
	// 	args: args{
	// 		output: []byte("1234 python script.py\n5678 java -jar myapp.jar\n"),
	// 		filterClassName: "",
	// 	},
	// 	want: []ProcessInfo{
	// 		{PID: 5678, ClassName: "myapp.jar"},
	// 	},
	// 	wantErr: false,
	// },
	// {
	// 	name: "Invalid PID",
	// 	args: args{
	// 		output: []byte("abcd java -jar myapp.jar\n"),
	// 		filterClassName: "",
	// 	},
	// 	want:    []ProcessInfo{},
	// 	wantErr: false,
	// },

	questions, _ := ReadFileToSlice("pssample.txt", true)
	answers, _ := ReadFileToSlice("pssample-answers.txt", true)
	for i := 0; i < len(questions); i++ {
		splits := strings.Split(answers[i], " ")
		pid, err := strconv.Atoi(splits[0])
		if err != nil {
			tests = append(tests, testStruct{
				name: fmt.Sprintf("line %d", i),
				args: args{
					output:          []byte(questions[i]),
					filterClassName: "",
				},
				want:    []ProcessInfo{},
				wantErr: false,
			})

			continue
		}

		tests = append(tests, testStruct{
			name: fmt.Sprintf("line %d", i),
			args: args{
				output:          []byte(questions[i]),
				filterClassName: "",
			},
			want:    []ProcessInfo{{PID: pid, ClassName: splits[1]}},
			wantErr: false,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetJavaProcessesFromBytes(tt.args.output, tt.args.filterClassName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetJavaProcessesFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetJavaProcessesFromBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
