package alfredo

import (
	"fmt"
	"strings"
)

// GitBranch will be injected with the current git branch name
var GitRevision string
var GitBranch string
var GitVersion string
var GitTimestamp string
var GitProduction string

func buildVersion(gitbranch string, gitversion string, gittimestamp string, mainbranch string, production bool) string {
	GitBranch = TrimQuotes(gitbranch)
	GitVersion = TrimQuotes(gitversion)
	GitTimestamp = TrimQuotes(gittimestamp)
	GitProduction = fmt.Sprintf("%t", production)

	VerbosePrintln("gitbranch=" + GitBranch)
	VerbosePrintln("ver=" + GitVersion)
	VerbosePrintln("time=" + GitTimestamp)
	VerbosePrintf("prod=%s", GitProduction)

	var gb string
	//fmt.Printf("Comparing %q vs %q\n", GitBranch, mainbranch)
	if strings.EqualFold(GitBranch, mainbranch) || production {
		gb = ""
	} else {
		gb = "-" + GitBranch
	}

	return fmt.Sprintf("%s%s (%s)\n", GitVersion, gb, GitTimestamp)
}

func BuildVersion() string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, "main", true)
}

func BuildVersionWithMainBranch(mainbranch string) string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, mainbranch, strings.EqualFold(strings.ToLower(GitProduction), "true"))
}
