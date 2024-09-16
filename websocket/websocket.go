package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cmd184psu/alfredo"
	"github.com/gorilla/websocket"
)

var jserver alfredo.JwtHttpsServerStruct

func main() {
	jserver.Init(8080)
	//jserver.EnableSSL("server.crt","server.key")
	jserver.Router.HandleFunc("/ws", handleWebSocket)

	if err := jserver.StartServer(); err != nil {
		panic(err.Error())
	}
}

// func main() {
// 	http.HandleFunc("/ws", handleWebSocket)
// 	log.Println("Server starting on :8080")
// 	log.Fatal(http.ListenAndServe(":8080", nil))
// }

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("Received WebSocket connection request")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	log.Println("WebSocket connection upgraded successfully")

	defer func() {
		log.Println("Closing WebSocket connection")
		conn.Close()
	}()

	// Send a test message immediately after connection
	err = conn.WriteMessage(websocket.TextMessage, []byte("Connection established successfully"))
	if err != nil {
		log.Println("Error sending test message:", err)
		return
	}
	log.Println("Sent test message to client")

	tailLog2(conn, "/opt/builderopt/alfredo/websocket/blah.log")

	// Keep the connection alive
	// for {
	// 	_, _, err := conn.ReadMessage()
	// 	if err != nil {
	// 		log.Println("Error reading message:", err)
	// 		break
	// 	}
	// }
}

// func tailLog(conn *websocket.Conn, logFile string) {
// 	//	logFile := "/opt/builderopt/alfredo/websocket/blah.log" // Replace with your log file path
// 	log.Printf("Attempting to open log file: %s", logFile)

// 	file, err := os.Open(logFile)
// 	if err != nil {
// 		log.Println("Error opening file:", err)
// 		return
// 	}
// 	defer file.Close()

// 	log.Println("Log file opened successfully")

// 	// Seek to the end of the file
// 	_, err = file.Seek(0, 2)
// 	if err != nil {
// 		log.Println("Error seeking file:", err)
// 		return
// 	}

// 	log.Println("Seeked to end of file")

// 	scanner := bufio.NewScanner(file)
// 	for {
// 		for scanner.Scan() {
// 			line := scanner.Text()
// 			log.Printf("Read line: %s", line)
// 			err := conn.WriteMessage(websocket.TextMessage, []byte(line))
// 			if err != nil {
// 				log.Println("Error writing message:", err)
// 				return
// 			}
// 			log.Println("Sent line to client")
// 		}
// 		if err := scanner.Err(); err != nil {
// 			log.Println("Error scanning file:", err)
// 			return
// 		}
// 		time.Sleep(time.Second) // Wait before checking for new lines
// 		log.Println("Checking for new lines...")
// 	}
// }

func tailLog2(conn *websocket.Conn, filePath string) {
	log.Printf("tailLog2(...,%s)", filePath)
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
		log.Printf("---")
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
				log.Printf("Read line: %s", line)
				err = conn.WriteMessage(websocket.TextMessage, []byte(line))
				if err != nil {
					log.Println("Error writing message:", err)
					return
				}
				log.Println("Sent line to client")

			}
			initialSize = currentSize
		}

		// Sleep for a short duration before checking again
		time.Sleep(5 * time.Second)
	}

}

func tailLog3(conn *websocket.Conn, filePath string) {
	log.Printf("tailLog3(...,%s)", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Get the last 5 lines initially
	lines, err := tailLastNLines(file, 5)
	if err != nil {
		log.Println("Error reading last 5 lines:", err)
		return
	}

	for _, line := range lines {
		log.Printf("Initial line: %s", line)
		err = conn.WriteMessage(websocket.TextMessage, []byte(line+"\n"))
		if err != nil {
			log.Println("Error writing message:", err)
			return
		}
	}

	// Start from the end of the file for new data
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		log.Println("Error seeking file:", err)
		return
	}

	for {
		log.Println("--waiting for changes--")
		currentSize, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			log.Println("Error getting file size:", err)
			return
		}

		if currentSize > offset {
			// Read new lines
			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				log.Printf("Read line: %s", line)
				err = conn.WriteMessage(websocket.TextMessage, []byte(line))
				if err != nil {
					log.Println("Error writing message:", err)
					return
				}
				log.Println("Sent line to client")
			}
			offset = currentSize
		}

		time.Sleep(5 * time.Second)
	}
}

func tailLastNLines(file *os.File, n int) ([]string, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	offset := int64(0)
	var lines []string

	for {
		if size <= 0 || len(lines) >= n {
			break
		}

		if size < 1024 {
			offset = -size
		} else {
			offset = -1024
		}

		file.Seek(offset, os.SEEK_END)
		reader := bufio.NewReader(file)
		partLines := []string{}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			partLines = append([]string{line}, partLines...)
			if len(partLines) > n {
				partLines = partLines[:n]
			}
		}
		lines = append(partLines, lines...)

		size += offset
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines, nil
}
