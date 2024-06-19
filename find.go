package alfredo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type INodeType int64

const (
	AllInodes INodeType = iota
	RegFileInodes
	DirectoryInodes
	SymlinkInodes
)

func GetFiletype(t string) INodeType {
	if strings.EqualFold(t, RegFileInodes.String()) {
		return RegFileInodes
	}
	if strings.EqualFold(t, SymlinkInodes.String()) {
		return SymlinkInodes
	}
	if strings.EqualFold(t, DirectoryInodes.String()) {
		return DirectoryInodes
	}
	return AllInodes
}

func (i INodeType) String() string {
	if i == RegFileInodes {
		return "f"
	}
	if i == DirectoryInodes {
		return "d"
	}
	if i == SymlinkInodes {
		return "l"
	}
	return "a"
}

func GetFindFileCLI(root string, pattern string, inType INodeType) string {
	if inType != AllInodes {
		return fmt.Sprintf("find %s -iname %q -type %q", root, pattern, inType.String())
	}
	return fmt.Sprintf("find %s -iname %q", root, pattern)
}

func FindFiles(root string, pattern string, inType INodeType) []string {
	var fileArray []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the file matches the pattern
		matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(info.Name()))
		if err != nil {
			return err
		}
		if matched {
			switch inType {
			case AllInodes:
				fileArray = append(fileArray, path)
			case RegFileInodes:
				if info.Mode().IsRegular() {
					fileArray = append(fileArray, path)
				}
			case DirectoryInodes:
				if info.IsDir() {
					fileArray = append(fileArray, path)
				}
			case SymlinkInodes:
				if !info.Mode().IsRegular() {
					fileArray = append(fileArray, path)
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
	return fileArray
}

func RemoveFiles(directoryPath string, prefix string) error {
	files := FindFiles(directoryPath, prefix, RegFileInodes)

	for i := 0; i < len(files); i++ {
		if strings.EqualFold(strings.TrimSpace(files[i]), "") {
			return errors.New("blank file in list")
		}
		if err := RemoveFile(files[i]); err != nil {
			return err
		}
	}
	return nil
}

func GetBaseName(f string) string {
	list := strings.Split(f, "/")
	return list[len(list)-1]
}
