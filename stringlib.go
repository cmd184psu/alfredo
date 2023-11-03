package alfredo

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}
func CSVtoArray(tagcsv string) []string {
	var tagcsv_array []string
	if strings.Contains(tagcsv, ",") {
		return strings.Split(tagcsv, ",")
	} else if len(tagcsv) > 0 {
		tagcsv_array = make([]string, 1)
		tagcsv_array[0] = tagcsv
	} else {
		tagcsv_array = make([]string, 0)
	}
	return tagcsv_array
}

func SliceContains(haystack []string, needle string) bool {
	sanitized_needle := strings.TrimSpace(strings.ToLower(needle))
	for _, h := range haystack {
		if strings.Contains(sanitized_needle, strings.TrimSpace(strings.ToLower(h))) {
			return true
		}
	}
	return false
}

func EmptyString(s string) bool {
	return len(s) == 0
}

func TrueIsYes(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func RemoveTag(l []string, s string) []string {
	for i := 0; i < len(l); i++ {
		if s == l[i] {
			l[i] = l[len(l)-1]
			return l[:len(l)-1]
		}
	}
	return l
}

func UniqTagSet(l []string) []string {
	var unique []string
	for _, v := range l {
		skip := false
		for _, u := range unique {
			if v == u {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, v)
		}
	}
	return unique
}

func HighLight(a string, b string, hl string) string {
	if strings.EqualFold(a, b) {
		return hl
	}
	return ""
}

func HumanReadableSeconds(s int64) string {
	d := time.Second * time.Duration(s)
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func HumanReadableStorageCapacity(b int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
		PB = 1 << 50
	)

	if b < KB {
		return fmt.Sprintf("%d B", b)
	}
	if b < MB {
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(KB))
	}
	if b < GB {
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(MB))
	}
	if b < TB {
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(GB))
	}
	if b < PB {
		return fmt.Sprintf("%.2f TiB", float64(b)/float64(TB))
	}
	return fmt.Sprintf("%.2f PiB", float64(b)/float64(PB))
}

func HumanReadableBigNumber(n int64) string {
	numStr := strconv.FormatInt(n, 10)
	var formattedStr string
	for i, digit := range numStr {
		if i > 0 && (len(numStr)-i)%3 == 0 {
			formattedStr += ","
		}
		formattedStr += string(digit)
	}
	return formattedStr
}

func TrimQuotes(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

func SetStringIfNotSet(v string, d string) string {
	if len(v) == 0 {
		return d
	}
	return v
}

func BlankIsNA(s string) string {
	if len(s) == 0 {
		return "N/A"
	}
	return s
}

func NotBlankIsMasked(s string) string {
	if len(s) == 0 {
		return s
	}
	return "********"
}
