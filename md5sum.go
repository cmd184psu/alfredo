package alfredo

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"

	//	"flag"
	"fmt"
)

func MD5sumFile(filePath string) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	//Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	//Tell the program to call the following function when the current function returns
	defer file.Close()

	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil

}

const (
	bufferSize = int64(8 * 1024 * 1024)
)

// work in progress; want to return string of hash for a partial file
func MD5sumChunk(filePath string, chunkSize int64) {

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		println("Cannot find file,", filePath, ", error:", err.Error())
		os.Exit(1)
	}

	chunkCount := int64(1)
	if chunkSize > 0 {
		chunkCount = (fileInfo.Size() / chunkSize) + 1
	} else {
		chunkSize = fileInfo.Size()
	}

	file, err := os.Open(filePath)
	if err != nil {
		println("Cannot open file,", filePath, ", error:", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	hash := md5.New()
	buff := make([]byte, bufferSize)
	for i := int64(0); i < chunkCount; i++ {
		hash.Reset()
		for chunkReaded := int64(0); chunkReaded < chunkSize; {
			m := bufferSize
			if chunkReaded+m > chunkSize {
				m = chunkSize - chunkReaded
			}

			n, err := file.Read(buff[0:m])
			if err != nil {
				if err == io.EOF {
					break
				}

				println("Cannot read file,", filePath, ", error:", err.Error())
				os.Exit(1)
			}
			hash.Write(buff[0:n])
			chunkReaded += int64(n)
		}

		if chunkCount > 1 {
			fmt.Printf("Chunk[%d] md5 = %x\r\n", i, hash.Sum(nil))
		} else {
			fmt.Printf("md5 = %x\r\n", hash.Sum(nil))
		}
	}
}
