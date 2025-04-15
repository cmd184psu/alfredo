package alfredo

import (
	"os"
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
