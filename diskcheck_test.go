package alfredo

import (
	"strings"
	"testing"
)

// helper to parse a single line
func mustParseMountLine(t *testing.T, line string) *mountInfo {
	t.Helper()
	mi, err := parseMountInfoLine(line)
	if err != nil {
		t.Fatalf("parseMountInfoLine error: %v", err)
	}
	return mi
}

func TestParseMountInfoLine_Basic(t *testing.T) {
	line := `26 25 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro`
	mi := mustParseMountLine(t, line)

	if mi.DevMajor != 8 || mi.DevMinor != 1 {
		t.Fatalf("unexpected dev major/minor: %d:%d", mi.DevMajor, mi.DevMinor)
	}
	if mi.Mount != "/" {
		t.Fatalf("unexpected mountpoint: %q", mi.Mount)
	}
	if mi.FsType != "ext4" {
		t.Fatalf("unexpected fstype: %q", mi.FsType)
	}
}

func TestFindMountForPathFromReader_RockyLinux(t *testing.T) {
	// Your actual Rocky Linux 9 lines + simulated /opt/decommission (same FS)
	mountinfo := `
21 98 0:20 / /sys rw,nosuid,nodev,noexec,relatime shared:2 - sysfs sysfs rw
22 98 0:5 / /proc rw,nosuid,nodev,noexec,relatime shared:26 - proc proc rw
98 1 8:4 / / rw,relatime shared:1 - ext4 /dev/sda4 rw
`
	r := strings.NewReader(mountinfo)

	// Test / finds root mount 8:4
	miRoot, err := findMountForPathFromReader(r, "/")
	if err != nil {
		t.Fatalf("findMountForPathFromReader(/) error: %v", err)
	}
	if miRoot.Mount != "/" || miRoot.DevMajor != 8 || miRoot.DevMinor != 4 {
		t.Fatalf("expected / mount 8:4, got %q %d:%d", miRoot.Mount, miRoot.DevMajor, miRoot.DevMinor)
	}

	// Fresh reader for /opt/decommission (should match same root mount)
	r2 := strings.NewReader(mountinfo)
	miOpt, err := findMountForPathFromReader(r2, "/opt/decommission")
	if err != nil {
		t.Fatalf("/opt/decommission lookup error: %v", err)
	}
	if miOpt.Mount != "/" || miOpt.DevMajor != 8 || miOpt.DevMinor != 4 {
		t.Fatalf("expected / mount 8:4 for /opt/decommission, got %q %d:%d", miOpt.Mount, miOpt.DevMajor, miOpt.DevMinor)
	}

	// Verify same FS decision
	if !sameUnderlyingFS(miRoot, miOpt) {
		t.Fatal("expected / and /opt/decommission to be same underlying FS")
	}
}



func TestFindMountForPathFromReader_LongestPrefix(t *testing.T) {
	// Simulate mountinfo where:
	//  - "/" is on 8:1
	//  - "/opt" is bind on same FS (8:1)
	//  - "/data" is different FS (8:2)
	mountinfo := `
26 25 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro
27 26 8:1 /opt /opt rw,relatime shared:1 - ext4 /dev/sda1 rw,bind
28 26 8:2 / /data rw,relatime shared:2 - ext4 /dev/sdb1 rw,errors=remount-ro
`

	// Create fresh readers for each call (simpler, more reliable)
	r1 := strings.NewReader(mountinfo)
	miRoot, err := findMountForPathFromReader(r1, "/")
	if err != nil {
		t.Fatalf("findMountForPathFromReader(/) error: %v", err)
	}
	if miRoot.Mount != "/" {
		t.Fatalf("expected / mount, got %q", miRoot.Mount)
	}
	if miRoot.DevMajor != 8 || miRoot.DevMinor != 1 {
		t.Fatalf("expected 8:1 for /, got %d:%d", miRoot.DevMajor, miRoot.DevMinor)
	}

	r2 := strings.NewReader(mountinfo) // Fresh reader
	miOpt, err := findMountForPathFromReader(r2, "/opt/something")
	if err != nil {
		t.Fatalf("findMountForPathFromReader(/opt/something) error: %v", err)
	}
	if miOpt.Mount != "/opt" {
		t.Fatalf("expected /opt mount, got %q", miOpt.Mount)
	}
	if miOpt.DevMajor != 8 || miOpt.DevMinor != 1 {
		t.Fatalf("expected 8:1 for /opt, got %d:%d", miOpt.DevMajor, miOpt.DevMinor)
	}

	// Verify sameUnderlyingFS works
	if !sameUnderlyingFS(miRoot, miOpt) {
		t.Fatal("expected / and /opt to be same underlying FS")
	}
}

func TestSameUnderlyingFS(t *testing.T) {
	a := &mountInfo{DevMajor: 8, DevMinor: 1}
	b := &mountInfo{DevMajor: 8, DevMinor: 1}
	c := &mountInfo{DevMajor: 8, DevMinor: 2}

	if !sameUnderlyingFS(a, b) {
		t.Fatalf("expected a and b to be same FS")
	}
	if sameUnderlyingFS(a, c) {
		t.Fatalf("expected a and c to be different FS")
	}
}

//invalid test
// func TestFindMountForPathFromReader_NoMatch(t *testing.T) {
// 	mountinfo := `
// 26 25 8:1 / / rw,relatime shared:1 - ext4 /dev/sda1 rw,errors=remount-ro
// `
// 	r := strings.NewReader(strings.TrimSpace(mountinfo) + "\n")

// 	// Now fails because /nonexistent/path doesn't have a dedicated mount
// 	if _, err := findMountForPathFromReader(r, "/nonexistent/path"); err == nil {
// 		t.Fatalf("expected error for path with no dedicated mount")
// 	}
// }
