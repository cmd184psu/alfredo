package alfredo

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ProcessInfo holds PID and className
type ProcessInfo struct {
	PID       int    `json:"pid"`
	ClassName string `json:"className"`
}

func JPS() ([]ProcessInfo, error) {
	return GetJavaProcesses("")
}

func GetJavaProcesses(filterClassName string) ([]ProcessInfo, error) {
	// Execute 'jps' command to list Java processes
	cmd := exec.Command("ps", "-eo pid,command")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return GetJavaProcessesFromBytes(output, filterClassName)
}

// getJavaProcesses returns a slice of ProcessInfo, optionally filtered by className
// sample line looks like this:
// 65056 /bin/java -Xmx2g -Xmn400m -Dlog4j.configurationFile=log4j-bucket-tools.xml com.cloudian.tools.bucket.BucketMigration -c md -mode rebuild -ip 192.168.1.31 -tmp /opt/migration/_tmp_ -p migration.properties.bucket1 -retry n -valid off -quit n
// 33423 /bin/java -Xmx2g -Xmn400m --add-opens java.base/java.math=ALL-UNNAMED --add-opens java.base/java.time=ALL-UNNAMED --add-opens java.base/java.lang=ALL-UNNAMED --add-opens java.base/java.util.concurrent=ALLUNNAMED --add-opens java.base/jdk.internal.misc=ALL-UNNAMED -XX:+EnableDynamicAgentLoading -Dlog4j.configurationFile=log4j-bucket-tools.xml com.cloudian.tools.bucket.BucketMigration -mode collect -c ver -tmp /opt/migration/_tmp_ -ip 192.168.1.31 -p migration.properties.versioning-bucket -retry n -v 3 -valid off -quit n
func GetJavaProcessesFromBytes(output []byte, filterClassName string) ([]ProcessInfo, error) {
	// Execute 'ps' command to list processes
	if len(output) == 0 {
		return []ProcessInfo{}, nil
	}
	var processes []ProcessInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	// Regex to match java processes and extract PID and className
	// Example command line: java -jar myapp.jar
	// or: java -Xmx1024m -cp ... com.example.MainClass
	javaRegex := regexp.MustCompile(`^\s*(\d+)\s+.*\bjava\b.*`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := javaRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			pidStr := matches[1]
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}
			commandLine := line[strings.Index(line, pidStr)+len(pidStr):]
			className := parseClassName(commandLine)
			//fmt.Printf("pid=%d, classname:%s\n", pid, className)
			if className != "" {
				VerbosePrintf("comparing className: %s with %s", className, filterClassName)
				if filterClassName == "" || strings.Contains(className, filterClassName) {
					VerbosePrintln("\tappending")
					processes = append(processes, ProcessInfo{PID: pid, ClassName: className})
				}
			}
		}
	}
	return processes, nil
}

// parseClassName extracts the className from the command line
func parseClassName(cmdLine string) string {
	// Look for the main class name pattern
	// e.g., java ... com.example.MainClass
	parts := strings.Fields(cmdLine)
	for _, part := range parts {
		if !strings.Contains(part, ".jar") && strings.Contains(part, ".") && !strings.HasPrefix(part, "-") && !strings.Contains(part, "ALL-UNNAMED") {
			splits := strings.Split(part, ".")
			return splits[len(splits)-1]
		}
	}
	return ""
}

func RunJPSExample() {
	// Example usage: get all Java processes
	processes, err := GetJavaProcesses("")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, p := range processes {
		fmt.Printf("PID: %d, ClassName: %s\n", p.PID, p.ClassName)
	}

	// Example: filter by className containing 'Main'
	filtered, err := GetJavaProcesses("Main")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Filtered processes:")
	for _, p := range filtered {
		fmt.Printf("PID: %d, ClassName: %s\n", p.PID, p.ClassName)
	}
}
