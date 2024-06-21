// Example code for alfredo library
// (c) 2024 C Delezenski <cmd184psu@gmail.com>
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cmd184psu/alfredo"
)

// func partialMatch(pattern, str string) bool {
// 	return regexp.MustCompile(pattern).MatchString(str)
// }

// func SeekMatch(uniqueSet map[string]bool, seek string) bool {
// 	// Iterate over unique substrings and test for partial matches
// 	for _, unique := range uniqueSet {
// 		if partialMatch(seek, unique) {
// 			return true
// 		}
// 	}
// 	return false
// }

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

func GetCLI(root string, pattern string, inType INodeType) string {
	if inType != AllInodes {
		return fmt.Sprintf("find %s -iname %q -type %q", root, pattern, inType.String())
	}
	return fmt.Sprintf("find %s -iname %q", root, pattern)
}

func LocalFindFiles2(root string, pattern string, inType INodeType) []string {
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

// func LocalFindFiles(startingPath string, needle string) ([]string, error) {
// 	alfredo.VerbosePrintf("BEGIN::  find %s -iname %q", startingPath, needle)
// 	var fileArray []string

// 	//var f FilenameStruct
// 	//f.Parse(prefix)

// 	re := regexp.MustCompile(`\*([^*]+)\*`)

// 	// Find all matches
// 	matches := re.FindAllStringSubmatch(needle, -1)

// 	// Set to store unique substrings
// 	uniqueSet := make(map[string]bool)

// 	// Iterate over matches and store unique substrings
// 	for _, match := range matches {
// 		uniqueSet[match[1]] = true
// 	}

// 	err := filepath.Walk(startingPath, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		if !info.IsDir() && SeekMatch(uniqueSet, path) {

// 			//VerbosePrintln("not a directory and file extension is ." + glob)
// 			//VerbosePrintln("")
// 			// 	fmt.Sprintf("HasPrefix(%s,%s)=%s",
// 			// 		path,
// 			// 		directoryPath+"/"+prefix,
// 			// 		TrueIsYes(strings.HasPrefix(path, directoryPath+"/"+prefix))
// 			// 	)
// 			// )
// 			var t string
// 			if strings.EqualFold(directoryPath, ".") {
// 				t = oprefix
// 			} else {
// 				t = directoryPath + "/" + oprefix
// 			}

// 			if len(prefix) > 0 && HasBase(path, t) ||
// 				len(prefix) == 0 {
// 				VerbosePrintln("appending " + path)
// 				if strings.HasPrefix(path, "/") {
// 					fileArray = append(fileArray, path)
// 				} else {

// 					fileArray = append(fileArray, directoryPath+"/"+path)
// 				}
// 			} else {
// 				VerbosePrintf("failed condition: len(prefix:%s)==%d?=0, strings.HasBase(%s,%s)=%s",
// 					prefix,
// 					len(prefix),
// 					path,
// 					t,
// 					TrueIsYes(HasBase(path, t)))
// 			}
// 		}
// 		return nil
// 	})
// 	if len(fileArray) != 0 {
// 		VerbosePrintln("file array contains:")
// 	}
// 	for i := 0; i < len(fileArray); i++ {
// 		VerbosePrintln(fmt.Sprintf("i=%d, item=%s", i, fileArray[i]))
// 	}
// 	alfredo.VerbosePrintf("END::  find %s -iname %q", startingPath, needle)

// 	return fileArray, err
// }

func main() {
	if strings.EqualFold(strings.ToLower(os.Args[0]), "gofind") {
		fmt.Println("--- find program ----")
	}
	path := "."
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	prefix := "*"
	if len(os.Args) > 2 && strings.EqualFold(strings.ToLower(os.Args[2]), "-iname") {
		prefix = os.Args[3]
	}
	findType := AllInodes
	if len(os.Args) > 4 && strings.EqualFold(strings.ToLower(os.Args[3]), "-type") {
		fmt.Println("---starting anew")
		findType = GetFiletype(os.Args[4])
	}
	fmt.Println(GetCLI(path, prefix, findType))
	//fmt.Printf("find %s -iname %q%s\n", path, prefix, findType.String())
	fmt.Println("--------------------")
	list := LocalFindFiles2(path, prefix, findType)
	//fmt.Println("find ./ -iname \"prefix*\"")
	//fmt.Println("\tpath=./ ---> arg1")
	//fmt.Println("\tprefix* --> arg1 of iname")
	fmt.Println("------start of list ---------")
	for i := 0; i < len(list); i++ {
		fmt.Printf("\t%s\n", list[i])
	}
	fmt.Println("------end of list ---------")
	fmt.Println("test")
	alfredo.VerbosePrintln("test")
}
