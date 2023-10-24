package alfredo

import (
	"fmt"
	"io"
	"math/rand"
	"os"
)

type diskDuplicatorStruct struct {
	InputFilename    string
	OutputFilename   string
	seek             int64
	count            int64
	totalBytesCopied int64
	bs               int
	result           string
	errMsg           string
	err              error
}

func (this diskDuplicatorStruct) WithInputFilename(f string) diskDuplicatorStruct {
	this.InputFilename = f
	return this
}
func (this diskDuplicatorStruct) WithOutputFilename(f string) diskDuplicatorStruct {
	this.OutputFilename = f
	return this
}
func (this diskDuplicatorStruct) WithSeek(s int64) diskDuplicatorStruct {
	this.seek = s
	return this
}
func (this diskDuplicatorStruct) WithCount(c int64) diskDuplicatorStruct {
	this.count = c
	return this
}
func (this diskDuplicatorStruct) WithBlockSize(b int) diskDuplicatorStruct {
	this.bs = b
	return this
}
func (this diskDuplicatorStruct) GetResult() string {
	return this.result
}
func (this diskDuplicatorStruct) GetErrorMsg() string {
	return this.errMsg
}
func (this diskDuplicatorStruct) GetError() error {
	return this.err
}
func (this diskDuplicatorStruct) HasError() bool {
	return this.err != nil
}
func (this diskDuplicatorStruct) Init() diskDuplicatorStruct {
	this.InputFilename = ""
	this.OutputFilename = ""
	this.seek = 0
	this.count = 0
	this.bs = 512
	this.errMsg = ""
	this.err = nil
	this.totalBytesCopied = 0
	return this
}
func (this diskDuplicatorStruct) Reset() diskDuplicatorStruct {
	return this.Init()
}

// cli=fmt.Sprintf("dd if=%s of=%s seek=%d count=%d bs=%d",random_file,filename,culmBytes,random_bytes_per_seek,blocksize_default)
func (this diskDuplicatorStruct) String() string {
	return fmt.Sprintf("dd if=%s of=%s seek=%d count=%d bs=%d", this.InputFilename, this.OutputFilename, this.seek, this.count, this.bs)
}

// write a sparse file f of size, randomly between sizeMin and sizeMax, r random 1's in the 0's.  Seek for 0's
func WriteSparseFile(f string, sizeMin int, sizeMax int, r int) error {
	minSize := sizeMin * 1024 * 1024
	maxSize := sizeMax * 1024 * 1024

	fileSize := rand.Intn(maxSize-minSize+1) + minSize

	// Create a new file
	file, err := os.Create(f)
	if err != nil {
		return err
	}
	defer file.Close()
	var seekto int64
	var r2 int
	var b byte
	seekto = 0
	for i := 0; i < r; i++ {
		r2 = rand.Intn(80)
		b = byte(r2)
		_, err = file.Write([]byte{b})
		if err != nil {
			fmt.Printf("Error writing: %v\n", err)
			return err
		}
		seekto += int64(fileSize) / int64(r)
		// Seek to the desired file size, creating a sparse file
		_, err = file.Seek(int64(seekto), 0)
		if err != nil {
			fmt.Printf("Error seeking: %v\n", err)
			return err
		}
	}

	//fmt.Printf("Sparse file 'sparse_file.bin' created with size %d bytes.\n", fileSize)
	return nil
}

func (this diskDuplicatorStruct) dd() diskDuplicatorStruct {
	// Define command-line flags
	// // inputFileName := flag.String("if", "", "Input file")
	// // outputFileName := flag.String("of", "", "Output file")
	// // seek := flag.Int64("seek", 0, "Seek offset")
	// // count := flag.Int64("count", -1, "Count")
	// // bs := flag.Int("bs", 512, "Block size")

	// flag.Parse()

	// Check if input and output file names are provided
	// if *inputFileName == "" || *outputFileName == "" {
	//     fmt.Println("Usage: go-dd -if inputfile -of outputfile [options]")
	//     flag.PrintDefaults()
	//     return
	// }

	// Open the input file
	inputFile, err := os.Open(this.InputFilename)
	if err != nil {
		this.err = err
		this.errMsg = fmt.Sprintf("Error opening input file: %v\n", err)
		return this
	}
	defer inputFile.Close()

	// Create the output file with the specified seek offset
	outputFile, err := os.OpenFile(this.OutputFilename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		this.err = err
		this.errMsg = fmt.Sprintf("Error creating output file: %v\n", err)
		return this
	}
	defer outputFile.Close()

	// Seek to the specified offset
	_, err = outputFile.Seek(this.seek, io.SeekStart)
	if err != nil {
		this.err = err
		this.errMsg = fmt.Sprintf("Error seeking to offset: %v\n", err)
		return this
	}

	// Read and write data in blocks
	buffer := make([]byte, this.bs)
	totalBytesCopied := int64(0)
	for {
		bytesRead, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			this.err = err
			this.errMsg = fmt.Sprintf("Error reading input file: %v\n", err)
			return this
		}

		if bytesRead == 0 || (this.count >= 0 && totalBytesCopied >= this.count) {
			break
		}

		// Write the block to the output file (creating sparse holes)
		bytesWritten, err := outputFile.Write(buffer[:bytesRead])
		if err != nil {
			this.err = err
			this.errMsg = fmt.Sprintf("Error writing to output file: %v\n", err)
			return this
		}

		this.totalBytesCopied += int64(bytesWritten)
	}

	this.result = fmt.Sprintf("Copied %d bytes from %s to %s\n", totalBytesCopied, this.InputFilename, this.OutputFilename)
	return this
}
