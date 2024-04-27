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

func LocalFindFiles2(root string, pattern string) []string {
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
			fileArray = append(fileArray, path)

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

	fmt.Printf("find %s -iname %q\n", path, prefix)
	fmt.Println("--------------------")
	list := LocalFindFiles2(path, prefix)
	fmt.Println("find ./ -iname \"prefix*\"")
	fmt.Println("\tpath=./ ---> arg1")
	fmt.Println("\tprefix* --> arg1 of iname")
	fmt.Println("------start of list ---------")
	for i := 0; i < len(list); i++ {
		fmt.Printf("\t%s\n", list[i])
	}
	fmt.Println("------end of list ---------")
	fmt.Println("test")
	alfredo.VerbosePrintln("test")
}
