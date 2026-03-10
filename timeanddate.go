package alfredo

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// simple test of a 3 character string; month or not a month;  Not locale friendly!
func IsMonth(m string) bool {
	m = strings.ToLower(m)
	if len(m) != 3 {
		return false
	}
	//not locale friendly!
	validMonths := []string{
		"jan", "feb", "mar", "apr", "may", "jun",
		"jul", "aug", "sep", "oct", "nov", "dec",
	}

	// Check if the input matches any valid abbreviation
	for _, valid := range validMonths {
		if m == valid {
			return true
		}
	}
	return false
}

const (
	TIME_FORMAT_1 = "02Jan06-03:04PM"
)

func GetFormattedTime(fmt string) string {
	return time.Now().Format(fmt)
}

func GetFormattedTime1() string {
	return GetFormattedTime(TIME_FORMAT_1)
}

func GetFirstOfMonthTimestamp() string {
	// Get the current time in the local timezone
	now := time.Now()

	// Create a new time.Time for the first day of the current month at midnight
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Format the time as per the specified format
	return firstOfMonth.Format("2006-01-02T15:04:05.000Z")
}

// should stat the file and return the last modified time
func GetLastModifiedTime(localFile string) (time.Time, error) {

	// Get file info
	fileInfo, err := os.Stat(ExpandTilde(localFile))
	if err != nil {
		return time.Time{}, err
	}

	// Return the last modified time
	return fileInfo.ModTime(), nil
}

func GetFileSize(localFile string) (int64, error) {

	// Get file info
	fileInfo, err := os.Stat(ExpandTilde(localFile))
	if err != nil {
		return 0, err
	}

	// Return the file size in bytes
	return fileInfo.Size(), nil
}

func GetFileSizeEasy(localFile string) int64 {
	s, err := GetFileSize(localFile)
	if err != nil {
		panic(fmt.Sprintf("failed to get file size for %s: %v", localFile, err))
	}
	return s
}

func SecondsToTimestamp(seconds int64) string {
	// Convert seconds to time.Time
	date := time.Unix(seconds, 0)

	// Format the date and time components
	year := date.Year()
	month := fmt.Sprintf("%02d", int(date.Month()))
	day := fmt.Sprintf("%02d", date.Day())
	hours := fmt.Sprintf("%02d", date.Hour())
	minutes := fmt.Sprintf("%02d", date.Minute())
	secs := fmt.Sprintf("%02d", date.Second())

	// Return the formatted string
	return fmt.Sprintf("%d-%s-%s %s:%s:%s", year, month, day, hours, minutes, secs)
}

type EpochTime struct {
	time.Time
}

func (et *EpochTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == "" {
		return nil
	}

	// Remove quotes if string, parse as number
	s := strings.Trim(string(data), `"`)
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	*et = EpochTime{time.UnixMilli(ms)}
	return nil
}

func (et EpochTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(et.Time.UnixMilli())
}

func (et *EpochTime) SetTime(t time.Time) {
	et.Time = t
}
func (et *EpochTime) Now() {
	et.Time = time.Now()
}

func (et *EpochTime) GetTime() time.Time {
	return et.Time
}

func EpochTimeFromTime(t time.Time) EpochTime {
	return EpochTime{Time: t}
}
