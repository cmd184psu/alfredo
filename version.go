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

func buildVersion(gitbranch string, gitversion string, gittimestamp string, mainbranch string) string {
	GitBranch = TrimQuotes(gitbranch)
	GitVersion = TrimQuotes(gitversion)
	GitTimestamp = TrimQuotes(gittimestamp)
	VerbosePrintln("gitbranch=" + GitBranch)
	VerbosePrintln("ver=" + GitVersion)
	VerbosePrintln("time=" + GitTimestamp)

	var gb string
	//fmt.Printf("Comparing %q vs %q\n", GitBranch, mainbranch)
	VerbosePrintln("production = " + GitProduction)
	if strings.EqualFold(GitBranch, mainbranch) || !strings.EqualFold(GitProduction, "") {
		gb = ""
	} else {
		gb = "-" + GitBranch
	}

	return fmt.Sprintf("%s%s (%s)\n", GitVersion, gb, GitTimestamp)
}

func BuildVersion() string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, "main")
}

func BuildVersionWithMainBranch(mainbranch string) string {
	return buildVersion(GitBranch, GitVersion, GitTimestamp, mainbranch)
}
