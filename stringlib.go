package alfredo

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

func Sanitized(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func SliceContains(haystack []string, needle string) bool {
	sanitized_needle := Sanitized(needle)
	for _, h := range haystack {
		if strings.EqualFold(sanitized_needle, Sanitized(h)) {
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

func GetFirstLineFromFile(f string) string {
	if !FileExistsEasy(f) {
		panic(fmt.Sprintf("File not found: %s", f))
	}
	list, err := LoadFileIntoSlice(f)
	if err != nil {
		panic(fmt.Sprintf("Unable to load file into slice: %s", f))
	}
	if len(list) == 0 {
		panic(fmt.Sprintf("File was empty: %s", f))
	}
	return list[0]
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
	return fmt.Sprintf("%02d H:%02d M:%02d S", hours, minutes, seconds)
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

func HumanReadableTimeStamp(t int64) string {
	d := time.Unix(t, 0)
	return d.Format("2006-01-02 15:04:05 MST")
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

func MD5SumFile(s string) string {
	ex := NewCLIExecutor()
	return ex.HashFile(s)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomAlphanumString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func GetClassName(fields []string) string {
	for i, field := range fields {
		if field == "java" {
			// Iterate over subsequent fields to find the main class name
			for j := i + 1; j < len(fields); j++ {
				if !strings.HasPrefix(fields[j], "-") {
					// Skip over -D, -X, and other JVM options
					if !strings.HasPrefix(fields[j], "/") && !strings.Contains(fields[j], "=") {
						return fields[j]
					}
				}
			}
		}
	}
	return "Unknown"
}

// getLastSegment returns the last segment of a dot-separated class name
func GetClassNameLastSegment(className string) string {
	segments := strings.Split(className, ".")
	return segments[len(segments)-1]
}

type BoolMapContainer struct {
	bmc map[string]bool
}

func (m *BoolMapContainer) FromCSV(itemsCSV string, s bool) {
	if len(itemsCSV) == 0 {
		return
	}

	reader := csv.NewReader(strings.NewReader(itemsCSV))
	records, _ := reader.ReadAll()

	if len(records) == 0 || len(records[0]) == 0 {
		return
	}

	for _, item := range records[0] {
		if s {
			m.Enable(item)
		} else {
			delete(m.bmc, item)
		}
	}
}

func (m *BoolMapContainer) EnableItems(itemsCSV string) {
	m.FromCSV(itemsCSV, true)
}
func (m *BoolMapContainer) DisableItems(itemsCSV string) {
	m.FromCSV(itemsCSV, false)
}
func (m BoolMapContainer) IsEnabled(key string) bool {
	value, exists := m.bmc[key]
	return exists && value
}
func (m *BoolMapContainer) Enable(key string) {
	if len(key) > 0 {
		m.bmc[key] = true
	}
}
func (m *BoolMapContainer) Disable(key string) {
	delete(m.bmc, key)
}

func (m BoolMapContainer) ToSlice() []string {
	var enabledItems []string
	for key, value := range m.bmc {
		if value {
			enabledItems = append(enabledItems, key)
		}
	}
	return enabledItems
}
func (m BoolMapContainer) ToCSV() string {
	return strings.Join(m.ToSlice(), ",")
}

func PrintSortedMap[K comparable, V any](m map[K]V) {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})

	for _, k := range keys {
		fmt.Printf("%v: %v\n", k, m[k])
	}
}

func MapToTableSlice(data map[string]string) []string {
	var content []string

	// Find the maximum length of keys and values
	maxKeyLen, maxValueLen := 0, 0
	for k, v := range data {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
		if len(v) > maxValueLen {
			maxValueLen = len(v)
		}
	}

	// Calculate total width of the table
	totalWidth := maxKeyLen + maxValueLen + 7 // 7 accounts for borders and spaces

	// Append top border
	content = append(content, "+"+strings.Repeat("-", totalWidth-2)+"+")

	// Append header
	content = append(content, fmt.Sprintf("| %-*s | %-*s |", maxKeyLen, "Field", maxValueLen, "Value"))

	// Append separator
	content = append(content, "+"+strings.Repeat("-", maxKeyLen+2)+"+"+strings.Repeat("-", maxValueLen+2)+"+")

	// Append data rows
	for k, v := range data {
		content = append(content, fmt.Sprintf("| %-*s | %-*s |", maxKeyLen, k, maxValueLen, v))
	}

	// Append bottom border
	content = append(content, "+"+strings.Repeat("-", totalWidth-2)+"+")

	return content
}

func MapToTableSliceOrdered(data map[string]string, order []string) []string {
	var content []string

	// Find the maximum length of keys and values
	maxKeyLen, maxValueLen := 0, 0
	for _, k := range order {
		v := data[k]
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
		if len(v) > maxValueLen {
			maxValueLen = len(v)
		}
	}

	// Calculate total width of the table
	totalWidth := maxKeyLen + maxValueLen + 7 // 7 accounts for borders and spaces

	// Append top border
	content = append(content, "+"+strings.Repeat("-", totalWidth-2)+"+")

	// Append header
	//content = append(content, fmt.Sprintf("| %-*s | %-*s |", maxKeyLen, "Field", maxValueLen, "Value"))

	// Append separator
	//content = append(content, "+"+strings.Repeat("-", maxKeyLen+2)+"+"+strings.Repeat("-", maxValueLen+2)+"+")

	// Append data rows in specified order
	for _, k := range order {
		v := data[k]
		content = append(content, fmt.Sprintf("| %-*s | %-*s |", maxKeyLen, k, maxValueLen, v))
	}

	// Append bottom border
	content = append(content, "+"+strings.Repeat("-", totalWidth-2)+"+")

	return content
}

// Tail reads the last numLines from the file, ignoring Java stack trace lines.
func Tail(filePath string, numLines int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	const chunkSize = 1024 // Read in chunks of 1KB
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := stat.Size()
	var lines []string
	var buffer []byte

	// Start reading from the end
	offset := fileSize
	for offset > 0 && len(lines) <= numLines {
		readSize := chunkSize
		if offset < int64(chunkSize) {
			readSize = int(offset)
		}
		offset -= int64(readSize)

		// Seek and read the chunk
		_, err := file.Seek(offset, io.SeekStart) // Updated to io.SeekStart
		if err != nil {
			return nil, err
		}

		chunk := make([]byte, readSize)
		_, err = file.Read(chunk)
		if err != nil {
			return nil, err
		}

		// Prepend the new chunk to buffer
		buffer = append(chunk, buffer...)
		lines = filterLines(splitLines(buffer))

		// Keep only the last `numLines` lines
		if len(lines) > numLines {
			lines = lines[len(lines)-numLines:]
		}
	}

	return lines, nil
}

// splitLines splits the input byte slice into lines.
func splitLines(data []byte) []string {
	scanner := bufio.NewScanner(bufio.NewReader(bytes.NewReader(data)))
	var result []string
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result
}

// filterLines removes lines that look like part of a Java stack trace.
func filterLines(lines []string) []string {
	stackTracePattern := regexp.MustCompile(`^\s*at\s+.*\((.*\.java:\d+|Native Method)\)`)
	filtered := []string{}
	for _, line := range lines {
		if !stackTracePattern.MatchString(line) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func EatErrorReturnString(s string, e error) string {
	if e == nil {
		return s
	}
	panic(e)
}

func DeepCopyStringToPointer(s string) *string {
	// Create a new string with the same value
	newString := s
	// Return the address of the new string
	return &newString
}

func PairToMap(k, v string) map[string]string {
	var m = make(map[string]string)

	if len(k) > 0 && len(v) > 0 {
		m[k] = v
	}
	return m
}

func StringContainsInCSV(s string, csv string) bool {
	if len(csv) == 0 {
		return false
	}
	for _, item := range strings.Split(csv, ",") {
		if len(item) == 0 {
			continue
		}
		if strings.Contains(s, item) {
			return true
		}
	}
	return false
}

func SliceToDoc(s []string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s[0]
	}
	return strings.Join(s[:len(s)-1], ", ") + " and " + s[len(s)-1]
}

func DiffStringBlobs(a, b string) string {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	return DiffStringContainers(aLines, bLines)
}

func TrimTrailingEmptyLines(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[:end]
}
func DiffStringContainers(aLines, bLines []string) string {
	var result strings.Builder
	aLines = TrimTrailingEmptyLines(aLines)
	bLines = TrimTrailingEmptyLines(bLines)

	for i := 0; i < len(aLines) || i < len(bLines); i++ {
		if i < len(aLines) && i < len(bLines) {
			if !strings.EqualFold(strings.TrimSpace(aLines[i]), strings.TrimSpace(bLines[i])) {
				result.WriteString(fmt.Sprintf("Line %d:\n- %s\n+ %s\n", i+1, aLines[i], bLines[i]))
			}
		} else if i < len(aLines) {
			result.WriteString(fmt.Sprintf("<Line %d:\n- %q\n", i+1, aLines[i]))
		} else {
			result.WriteString(fmt.Sprintf(">Line %d:\n+ %q\n", i+1, bLines[i]))
		}
	}
	return result.String()
}

func RemoveKeyPartialFromSlice(s string, slice []string) []string {
	newSlice := make([]string, 0)
	for _, v := range slice {
		if !strings.Contains(strings.ToLower(v), strings.ToLower(s)) {
			newSlice = append(newSlice, v)
			//			return append(slice[:i], slice[i+1:]...)
		}
	}
	return newSlice
}

func WriteLineToFile(filePath, line string) error {
	VerbosePrintf("write to file: %s", filePath)
	file, err := os.OpenFile(ExpandTilde(filePath), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(line + "\n")
	return err
}
