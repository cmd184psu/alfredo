package alfredo

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func TrimSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}
func CSVtoArray(tagcsv string) []string {
	var tagcsv_array []string
	if strings.Contains(tagcsv, ",") {
		return strings.Split(tagcsv, ",")
	} else if len(tagcsv) > 0 {
		tagcsv_array = make([]string, 1)
		tagcsv_array[0] = tagcsv
	} else {
		tagcsv_array = make([]string, 0)
	}
	return tagcsv_array
}

func SliceContains(haystack []string, needle string) bool {
	sanitized_needle := strings.TrimSpace(strings.ToLower(needle))
	for _, h := range haystack {
		if strings.Contains(sanitized_needle, strings.TrimSpace(strings.ToLower(h))) {
			return true
		}
	}
	return false
}

func GetFirstLineFromSlice(content string, needle string) string {
	list := strings.Split(content, "\n")
	for i := 0; i < len(list); i++ {
		if strings.HasPrefix(list[i], needle) {
			return list[i]
		}
	}
	return ""
}

func EmptyString(s string) bool {
	return len(s) == 0
}

func TrueIsYes(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func RemoveTag(l []string, s string) []string {
	for i := 0; i < len(l); i++ {
		if s == l[i] {
			l[i] = l[len(l)-1]
			return l[:len(l)-1]
		}
	}
	return l
}

func UniqTagSet(l []string) []string {
	var unique []string
	for _, v := range l {
		skip := false
		for _, u := range unique {
			if v == u {
				skip = true
				break
			}
		}
		if !skip {
			unique = append(unique, v)
		}
	}
	return unique
}

func HighLight(a string, b string, hl string) string {
	if strings.EqualFold(a, b) {
		return hl
	}
	return ""
}

func HumanReadableSeconds(s int64) string {
	d := time.Second * time.Duration(s)
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func HumanReadableStorageCapacity(b int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
		PB = 1 << 50
	)

	if b < KB {
		return fmt.Sprintf("%d B", b)
	}
	if b < MB {
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(KB))
	}
	if b < GB {
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(MB))
	}
	if b < TB {
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(GB))
	}
	if b < PB {
		return fmt.Sprintf("%.2f TiB", float64(b)/float64(TB))
	}
	return fmt.Sprintf("%.2f PiB", float64(b)/float64(PB))
}

func HumanReadableBigNumber(n int64) string {
	numStr := strconv.FormatInt(n, 10)
	var formattedStr string
	for i, digit := range numStr {
		if i > 0 && (len(numStr)-i)%3 == 0 {
			formattedStr += ","
		}
		formattedStr += string(digit)
	}
	return formattedStr
}

func TrimQuotes(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

func SetStringIfNotSet(v string, d string) string {
	if len(v) == 0 {
		return d
	}
	return v
}

func BlankIsNA(s string) string {
	if len(s) == 0 {
		return "N/A"
	}
	return s
}

func NotBlankIsMasked(s string) string {
	if len(s) == 0 {
		return s
	}
	return "********"
}

func ExpandTilde(f string) string {
	if !strings.HasPrefix(f, "~") {
		return f
	}
	if strings.HasPrefix(f, "~/") {
		return fmt.Sprintf("%s/%s", os.Getenv("HOME"), f[2:])
	}
	if strings.HasPrefix(f, "~") {
		dirs := strings.Split(os.Getenv("HOME"), "/")
		return fmt.Sprintf("/%s/%s", dirs[1], f[1:])
	}
	return f
}

func correctedParams(directoryPath *string, prefix *string, glob *string) {
	(*glob) = strings.TrimPrefix(*glob, "*")
	(*glob) = strings.TrimPrefix(*glob, ".")
	(*prefix) = strings.TrimPrefix(*prefix, "/")
	(*prefix) = strings.TrimSuffix(*prefix, "*")
	(*directoryPath) = strings.TrimSuffix(*directoryPath, "/")
}

func GetFileFindCLI(directoryPath string, prefix string, glob string) string {
	correctedParams(&directoryPath, &prefix, &glob)
	return fmt.Sprintf("find " + directoryPath + " -iname \"" + prefix + "*." + glob + "\"")
}

func MoveDirs(needleSuffix string, st int, target string) error {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			//VerbosePrintln(fmt.Sprintf("does %s have suffix needle %s?\n", path, needleSuffix))
			//VerbosePrintln("\tfound directory!")
			resolved := false
			if strings.HasSuffix(path, needleSuffix) {
				//VerbosePrintln("\tYES!")
				for i := st; i < 100; i++ {
					if !FileExistsEasy(fmt.Sprintf("%s-%d", target, i)) {
						st = i
						resolved = true
						break
					}
				}
				if !resolved {
					panic("unable to create directory")
				}

				fmt.Printf("mv -n %q \"%s-%d\"\n", path, target, st)
				st++
			}
			//  else {
			// 	VerbosePrintln("\tNO!")
			// }
		}
		return nil
	})
	return err
}

func MoveFiles(needleSuffix string, st int, target string) error {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, "@") {
			DebugPrintf("rejecting path %s", path)
			return nil
		}
		if strings.HasPrefix(path, ".git") {
			DebugPrintf("rejecting path %s", path)
			return nil
		}
		if err != nil {
			return err
		}
		var sfns, tfns FilenameStruct
		if !info.IsDir() {
			VerbosePrintln(fmt.Sprintf("does %s have suffix needle %s?\n", path, needleSuffix))
			VerbosePrintln("\tfound file!")
			tfns.Parse(target)

			resolved := false
			if strings.HasSuffix(path, needleSuffix) {
				//VerbosePrintln("\tYES!")
				for i := st; i < 100; i++ {
					proposedFile := fmt.Sprintf("%s-%d%s", tfns.GetFullName(), i, sfns.ext)
					VerbosePrintf("propsed new filename: %s", proposedFile)
					if !FileExistsEasy(proposedFile) {
						st = i
						resolved = true
						break
					}
				}
				if !resolved {
					panic("unable to create directory")
				}

				// # path=some/other/path/something.txt
				// # fns.base=st
				// # num=1
				// # fns.ext=
				// # GetFullName()=/home/cdelezenski/somewhereelse/st
				// # moving file "some/other/path/something.txt" to "st-1."
				// mv -nv "some/other/path/something.txt" "st-1."
				// :::: does testfile.txt have suffix needle something.txt?
				//  ::::
				// :::: 	found file! ::::
				// :::: alfredo::util.go::FilenameStruct::Parse(/home/cdelezenski/somewhereelse/st)=/home/cdelezenski/somewhereelse/st ::::
				// # Concluded movedirs
				// [cdelezenski@builder go-multiren]$ ./makemr && ./multiren-linux -filename something.txt -verbose -move -newpath /home/cdelezenski/somewhereelse/st
				sfns.Parse(path)
				CommentPrintf("path=%s", path) //found
				CommentPrintf("fns.base=%s", tfns.base)
				CommentPrintf("num=%d", st)
				CommentPrintf("fns.ext=%s", sfns.ext)
				CommentPrintf("GetFullName()=%s", tfns.GetFullName())

				CommentPrintf("moving file %q to \"%s-%d%s\"", sfns.GetFullName(), tfns.GetFullName(), st, sfns.ext)
				fmt.Printf("mv -nv %q \"%s-%d%s\"\n", sfns.GetFullName(), tfns.GetFullName(), st, sfns.ext)
				st++
			}
			//  else {
			// 	VerbosePrintln("\tNO!")
			// }
		}
		return nil
	})
	return err
}

func HasBase(filename string, base string) bool {
	var f FilenameStruct
	if err := f.Parse(filename); err != nil {
		panic(err.Error())
	}

	if strings.HasPrefix(base, "*") && strings.HasSuffix(base, "*") {
		return strings.Contains(f.GetBase(), base[1:(len(base)-1)])
	}
	if strings.HasPrefix(base, "*") {
		return strings.HasSuffix(f.GetBase(), base[1:])
	}
	VerbosePrintf("should get here: HasSuffix(%s,%s)=%s", f.GetBase(), base[:(len(base)-1)], TrueIsYes(strings.HasSuffix(base, "*")))
	if strings.HasSuffix(base, "*") {
		VerbosePrintf("\tRETURN=%s", TrueIsYes(strings.HasSuffix(f.GetBase(), base[:(len(base)-1)])))
		return strings.HasSuffix(f.GetBase(), base[:(len(base)-1)])
	}

	return strings.EqualFold(f.GetBase(), base)
}

func MD5SumBA(ba []byte) string {
	hash := md5.Sum(ba)
	return hex.EncodeToString(hash[:])
}

func MD5SumString(s string) string {
	return MD5SumBA([]byte(s))
}
