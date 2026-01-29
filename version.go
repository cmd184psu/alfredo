package alfredo

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// GitBranch will be injected with the current git branch name
var GitRevision string
var GitBranch string
var GitVersion string
var GitTimestamp string
var GitProduction string

const layout = "02Jan06-03:04pm"
const expirationDay = 365

func OccurredXDaysAgo(dateStr string, x int) bool {
	now := time.Now()
	if strings.HasSuffix(dateStr, "PM") {
		dateStr = strings.TrimSuffix(dateStr, "PM") + "pm"
	} else if strings.HasSuffix(dateStr, "AM") {
		dateStr = strings.TrimSuffix(dateStr, "AM") + "am"
	}
	parsed, err := time.ParseInLocation(layout, dateStr, now.Location())
	if err != nil {
		panic(err)
	}
	diff := now.Sub(parsed)
	days := int(diff.Hours() / 24)
	return days == x && now.After(parsed)
}

func WillOccurYDaysFromNow(dateStr string, y int) bool {
	now := time.Now()
	parsed, err := time.ParseInLocation(layout, dateStr, now.Location())
	if err != nil {
		panic(err)
	}
	diff := parsed.Sub(now)
	days := int(diff.Hours() / 24)
	return days == y && parsed.After(now)
}

// func DaysUntilExpiration(expirationDateStr string) (int, error) {
// 	// Parse the expiration date
// 	expirationDate, err := time.Parse(layout, expirationDateStr)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to parse date: %w", err)
// 	}

// 	// Get current time
// 	now := time.Now()

// 	// Calculate difference in hours and convert to days
// 	diff := expirationDate.Sub(now).Hours() / 24

// 	// Round to nearest integer
// 	return int(math.Round(diff)), nil
// }

func ParseCustomTimestamp(dateStr string) (time.Time, error) {
	// Use local time zone or specify one explicitly
	loc := time.Now().Location()
	parsedTime, err := time.ParseInLocation(layout, dateStr, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}
	return parsedTime, nil
}

func DaysUntilExpiration(compileTime time.Time, daysInFuture int) int {
	expiration := compileTime.AddDate(0, 0, daysInFuture)
	now := time.Now()
	diff := expiration.Sub(now).Hours() / 24
	return int(math.Ceil(diff))
}

func buildVersion(gitbranch string, gitversion string, gittimestamp string, mainbranch string, production bool) string {
	GitBranch = TrimQuotes(gitbranch)
	GitVersion = TrimQuotes(gitversion)
	GitTimestamp = TrimQuotes(gittimestamp)
	//	GitProduction = fmt.Sprintf("%t", production)

	production = strings.EqualFold(GitProduction, "true")

	VerbosePrintln("gitbranch=" + GitBranch)
	VerbosePrintln("ver=" + GitVersion)

	if strings.HasSuffix(GitTimestamp, "PM") {
		GitTimestamp = strings.TrimSuffix(GitTimestamp, "PM") + "pm"
	} else if strings.HasSuffix(GitTimestamp, "AM") {
		GitTimestamp = strings.TrimSuffix(GitTimestamp, "AM") + "am"
	}

	VerbosePrintln("time=" + GitTimestamp)
	//	VerbosePrintf("prod=%s", GitProduction)

	var gb string
	//fmt.Printf("Comparing %q vs %q\n", GitBranch, mainbranch)
	VerbosePrintln("production = " + GitProduction)
	if strings.EqualFold(GitBranch, mainbranch) || production {
		gb = ""
	} else {
		gb = "-" + GitBranch
	}

	if OccurredXDaysAgo(GitTimestamp, expirationDay) {
		panic("This version has expired, please update to a newer version.")
	}

	compiletime, err := ParseCustomTimestamp(GitTimestamp)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse GitTimestamp: %v", err))
	}
	if compiletime.IsZero() {
		panic("GitTimestamp is not set or is invalid.")
	}

	x := DaysUntilExpiration(compiletime, expirationDay)
	//if x < 5 {
	if !production {
		fmt.Fprintf(os.Stderr, "This version will expire in %d days.\n", x)
	}

	return fmt.Sprintf("%s%s (%s)\n", GitVersion, gb, GitTimestamp)
}

func BuildVersion() string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, "main", true)
}

func BuildVersionWithMainBranch(mainbranch string) string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, mainbranch, strings.EqualFold(strings.ToLower(GitProduction), "true"))
}
