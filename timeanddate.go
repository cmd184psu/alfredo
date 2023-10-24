package alfredo

import "strings"

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
