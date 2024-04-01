package alfredo

import (
	"fmt"
	"strings"
)

// GitBranch will be injected with the current git branch name
var GitBranch string
var GitVersion string
var GitTimestamp string

func BuildVersion(gitbranch string, gitversion string, gittimestamp string, mainbranch string) string {
	GitBranch = TrimQuotes(gitbranch)
	GitVersion = TrimQuotes(gitversion)
	GitTimestamp = TrimQuotes(gittimestamp)
	VerbosePrintln("gitbranch=" + GitBranch)
	VerbosePrintln("ver=" + GitVersion)
	VerbosePrintln("time=" + GitTimestamp)

	var gb string
	fmt.Printf("Comparing %q vs %q\n", GitBranch, mainbranch)
	if strings.EqualFold(GitBranch, mainbranch) {
		gb = ""
	} else {
		gb = "-" + GitBranch
	}

	return fmt.Sprintf("%s%s (%s)\n", GitVersion, gb, GitTimestamp)
}
