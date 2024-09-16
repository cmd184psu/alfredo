package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	// The file you want to watch
	filePath := "yourfile.txt"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Get the initial size of the file
	initialStat, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return
	}
	initialSize := initialStat.Size()

	for {
		// Get the current size of the file
		currentStat, err := file.Stat()
		if err != nil {
			fmt.Println("Error getting file info:", err)
			return
		}
		currentSize := currentStat.Size()

		// If the file has grown, read the new data
		if currentSize > initialSize {
			file.Seek(initialSize, 0)
			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				fmt.Print(line)
			}
			initialSize = currentSize
		}

		// Sleep for a short duration before checking again
		time.Sleep(1 * time.Second)
	}
}
