package alfredo

import (
	"fmt"
	"os"
	"strconv"
)

const sqlite_bin = "/usr/bin/sqlite3"

type DatabaseStruct struct {
	DbPath string `json:"db_path"`
	Table  string `json:"table"`
	exe    *CLIExecutor
}

type DatabaseConfig struct {
	DbPath  string `json:"db_path"`
	Table   string `json:"table"`
	SSHHost string `json:"ssh_host"`
	SSHKey  string `json:"ssh_key"`
	SSHUser string `json:"ssh_user"`
}

func (db *DatabaseStruct) LoadConfig(filePath string) error {
	config := DatabaseConfig{}
	if err := ReadStructFromJSONFile(filePath, &config); err != nil {
		return err
	}
	if len(config.SSHUser) == 0 {
		config.SSHUser = os.Getenv("USER")
	}

	db.WithDbPath(config.DbPath).
		WithTable(config.Table)
	db.exe.WithSSH(config.SSHHost, config.SSHKey, config.SSHUser)

	return nil
}

func NewSQLiteDB() *DatabaseStruct {
	db := &DatabaseStruct{}
	db.exe = NewCLIExecutor()
	db.exe.WithCaptureStdout(true).
		WithCommand(sqlite_bin, db.DbPath).
		WithTrimWhiteSpace(true)
	return db
}

func (db *DatabaseStruct) WithDbPath(path string) *DatabaseStruct {
	db.DbPath = path
	db.exe.WithCommand(sqlite_bin, db.DbPath)
	return db
}

func (db *DatabaseStruct) WithTable(table string) *DatabaseStruct {
	db.Table = table
	return db
}

func (db *DatabaseStruct) WithSSH(host string, key string) *DatabaseStruct {
	db.exe.sshHost = host
	db.exe.sshKey = key
	return db
}

func (db *DatabaseStruct) CLItarget() string {
	return fmt.Sprintf("%s \"%s\"", sqlite_bin, db.DbPath)
}

func (db *DatabaseStruct) QueryPayload(sel string, where string) string {
	var w string
	if len(where) != 0 {
		w = fmt.Sprintf(" WHERE %s", where)
	}
	return fmt.Sprintf("SELECT %s FROM %s%s;\n", sel, db.Table, w)
}

func (db *DatabaseStruct) CountPayload(c string, where string) string {
	return db.QueryPayload(fmt.Sprintf("COUNT(%s)", c), where)
}
func (db *DatabaseStruct) AvgPayload(c string, where string) string {
	return db.QueryPayload(fmt.Sprintf("AVG(%s)", c), where)
}
func (db *DatabaseStruct) SumPayload(c string, where string) string {
	return db.QueryPayload(fmt.Sprintf("SUM(%s)", c), where)
}

//sqlite3 your_database.db "UPDATE your_table SET state = 1 WHERE size > 1000 AND state = 0;"

func (db *DatabaseStruct) UpdatePayload(set string, where string) string {
	var w string
	if len(where) != 0 {
		w = fmt.Sprintf(" WHERE %s", where)
	}
	return fmt.Sprintf("UPDATE %s SET %s%s;\n", db.Table, set, w)
}
func (db *DatabaseStruct) DeletePayload(where string) string {
	var w string
	if len(where) == 0 {
		panic("unable to delete everything with this function; where was empty")
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s;\n", db.Table, w)
}

func (db *DatabaseStruct) Delete(where string) error {
	db.exe.WithRequestPayload(db.DeletePayload(where))
	return db.Execute()
}

func (db *DatabaseStruct) Execute() error {
	return db.exe.Execute()
}

func (db *DatabaseStruct) Query(query string) error {
	db.exe.WithRequestPayload(query)
	return db.Execute()
}

func (db *DatabaseStruct) GetPayload() string {
	return db.exe.GetRequestPayload()
}
func (db *DatabaseStruct) Count(sel string, where string) error {
	if len(sel) == 0 || sel == "*" {
		panic("Count requires a column name")
	}
	db.exe.WithRequestPayload(db.CountPayload(sel, where))
	return db.Execute()
}
func (db *DatabaseStruct) Sum(sel string, where string) error {
	if len(sel) == 0 || sel == "*" {
		panic("Sum requires a column name")
	}
	VerbosePrintf("payload is: %s", db.SumPayload(sel, where))

	db.exe.WithRequestPayload(db.SumPayload(sel, where))
	return db.Execute()
}

func (db *DatabaseStruct) Avg(sel string, where string) error {
	if len(sel) == 0 || sel == "*" {
		panic("Avg requires a column name")
	}

	db.exe.WithRequestPayload(db.AvgPayload(sel, where))
	return db.Execute()
}

func (db *DatabaseStruct) Update(set string, where string) error {
	db.exe.WithRequestPayload(db.UpdatePayload(set, where))
	return db.Execute()
}

func (db *DatabaseStruct) GetResult() string {
	return db.exe.GetResponseBody()
}

func (db *DatabaseStruct) GetResultInt() int {
	result, err := strconv.Atoi(db.exe.GetResponseBody())
	if err != nil {
		return 0
	}
	return result
}

func (db *DatabaseStruct) GetResultInt64() int64 {
	return int64(db.GetResultFloat())
}

func (db *DatabaseStruct) GetResultFloat() float64 {
	result, err := strconv.ParseFloat(db.exe.GetResponseBody(), 64)
	if err != nil {
		return 0.0
	}
	return result
}

func DbSelfTest(dbPath string, query string) error {
	if len(dbPath) == 0 {
		return fmt.Errorf("db path is empty")
	}
	db := NewSQLiteDB()
	db.WithDbPath(dbPath)

	fmt.Printf("query was %q\n", query)
	if err := db.Query(query); err != nil {
		panic(err)
		fmt.Println("error: ", err)
		return err
	}

	if err := db.exe.DumpOutput().Execute(); err != nil {
		panic(err)
		fmt.Println("error: ", err)
		return err
	}
	fmt.Println("result: ", db.exe.GetResponseBody())
	fmt.Println("status code: ", db.exe.GetStatusCode())
	return nil
}
