package alfredo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	OneKiB = 1 << 10      // 1024 bytes
	OneMiB = 1 << 20      // 1,048,576 bytes  
	OneGiB = 1 << 30      // 1,073,741,824 bytes
	OneTiB = 1 << 40      // 1,099,511,627,776 bytes
)

type mountInfo struct {
	DevMajor int
	DevMinor int
	Root     string
	Mount    string
	FsType   string
	Source   string
	Options  string
}

func CheckSpaceForPathUpToOneGiB(rootPath, bindPath string) bool {
	ok, err := CheckSpaceForPath(rootPath, bindPath, OneGiB)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
	return ok
}
func CheckSpaceForPathUpToOneTiB(rootPath, bindPath string) bool {
	ok, err := CheckSpaceForPath(rootPath, bindPath, OneTiB)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
	return ok
}

// CheckSpaceForPath checks whether the effective filesystem for bindPath
// (falling back to rootPath when they are the same FS) has at least
// thresholdBytes of free space.
func CheckSpaceForPath(rootPath, bindPath string, thresholdBytes uint64) (bool, error) {
	rootPath = filepath.Clean(rootPath)
	bindPath = filepath.Clean(bindPath)

	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return false, fmt.Errorf("open mountinfo: %w", err)
	}
	defer f.Close()

	rootMount, err := findMountForPathFromReader(f, rootPath)
	if err != nil {
		return false, fmt.Errorf("find mount for %s: %w", rootPath, err)
	}

	// Rewind reader for the second lookup.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("rewind mountinfo: %w", err)
	}

	bindMount, err := findMountForPathFromReader(f, bindPath)
	if err != nil {
		return false, fmt.Errorf("find mount for %s: %w", bindPath, err)
	}

	var pathToCheck string
	if sameUnderlyingFS(rootMount, bindMount) {
		pathToCheck = rootPath
	} else {
		pathToCheck = bindPath
	}

	free, err := freeBytes(pathToCheck)
	if err != nil {
		return false, fmt.Errorf("free space for %s: %w", pathToCheck, err)
	}

	return free >= thresholdBytes, nil
}

func freeBytes(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}

func sameUnderlyingFS(a, b *mountInfo) bool {
	return a.DevMajor == b.DevMajor && a.DevMinor == b.DevMinor
}

func findMountForPathFromReader(r io.Reader, path string) (*mountInfo, error) {
	path = filepath.Clean(path)
	
	var best *mountInfo
	bestLen := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		mi, err := parseMountInfoLine(line)
		if err != nil {
			continue
		}
		
		// Handle root mount "/" specially - don't clean it to "."
		mountPath := mi.Mount
		if mountPath == "/" {
			mountPath = "/"
		} else {
			mountPath = filepath.Clean(mi.Mount)
		}
		
		if !strings.HasPrefix(path, mountPath) {
			continue
		}
		if len(mountPath) > bestLen {
			bestLen = len(mountPath)
			cp := *mi
			best = &cp
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if best == nil {
		return nil, errors.New("no mount entry found for path")
	}
	return best, nil
}

func parseMountInfoLine(line string) (*mountInfo, error) {
	parts := strings.SplitN(line, " - ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mountinfo line (split failed): %q", line)
	}
//	fmt.Printf("DEBUG: parts[0]=%q\n", parts[0])  // TEMPORARY

	// Try parsing with flexible field handling
	fields := strings.Fields(parts[0])
//	fmt.Printf("DEBUG: fields=%v (len=%d)\n", fields, len(fields))  // TEMPORARY
	if len(fields) < 6 {
		return nil, fmt.Errorf("too few fields: %v", fields)
	}

	// Field 0=ID, 1=parentID, 2=major:minor, 3=root, 4=mountpoint, 5+=mountopts+optional
	devParts := strings.Split(fields[2], ":")
	if len(devParts) != 2 {
		return nil, fmt.Errorf("invalid major:minor in %q", fields[2])
	}
	maj, err := strconv.Atoi(devParts[0])
	if err != nil {
		return nil, fmt.Errorf("maj parse %q: %w", devParts[0], err)
	}
	min, err := strconv.Atoi(devParts[1])
	if err != nil {
		return nil, fmt.Errorf("min parse %q: %w", devParts[1], err)
	}

	root := fields[3]
	mount := fields[4]

	// Post-fields
	postFields := strings.Fields(parts[1])
	if len(postFields) < 2 {
		return nil, fmt.Errorf("post-fields too short: %q", parts[1])
	}
	fsType := postFields[0]
	source := postFields[1]

//	fmt.Printf("DEBUG: parsed maj:%d min:%d mount:%q fs:%q source:%q\n", maj, min, mount, fsType, source)  // TEMPORARY

	return &mountInfo{
		DevMajor: maj,
		DevMinor: min,
		Root:     root,
		Mount:    mount,
		FsType:   fsType,
		Source:   source,
		Options:  fields[5],
	}, nil
}

func parseMountInfoLineBROKEN(line string) (*mountInfo, error) {
	parts := strings.SplitN(line, " - ", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid mountinfo line")
	}
	fields := strings.Fields(parts[0])
	if len(fields) < 7 {
		return nil, errors.New("invalid mountinfo pre-fields")
	}

	devParts := strings.Split(fields[2], ":")
	if len(devParts) != 2 {
		return nil, errors.New("invalid major:minor")
	}
	maj, err := strconv.Atoi(devParts[0])
	if err != nil {
		return nil, err
	}
	min, err := strconv.Atoi(devParts[1])
	if err != nil {
		return nil, err
	}

	root := fields[3]
	mount := fields[4]
	opts := fields[5]

	postFields := strings.Fields(parts[1])
	if len(postFields) < 3 {
		return nil, errors.New("invalid mountinfo post-fields")
	}
	fsType := postFields[0]
	source := postFields[1]
	superOpts := strings.Join(postFields[2:], " ")

	return &mountInfo{
		DevMajor: maj,
		DevMinor: min,
		Root:     root,
		Mount:    mount,
		FsType:   fsType,
		Source:   source,
		Options:  opts + "," + superOpts,
	}, nil
}
