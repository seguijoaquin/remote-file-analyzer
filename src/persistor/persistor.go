package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Send is responsible for sending messages through a
// generic stream-oriented network connection
func send(conn *net.Conn, request []byte) error {
	var err error
	var n int
	n, err = (*conn).Write(request)

	for n < len(request) {
		n, err = (*conn).Write(request[n:])
		if err != nil {
			break
		}
	}
	return err
}

func scanSubDir(basePath string) []byte {
	fileSize := []byte("")
	c, err := ioutil.ReadDir(basePath)
	check(err)
	for _, entry := range c {
		fmt.Println(" ", entry.Name(), string(fileSize))
		if !entry.IsDir() {
			fileSize, _ = ioutil.ReadFile(filepath.Join(basePath, entry.Name()))
		} else {
			fileSize = scanSubDir(filepath.Join(basePath, entry.Name()))
		}
	}
	return fileSize
}

func handleRequest(conn *net.Conn, request persistorRequestDTO) error {
	basePath := filepath.Join("/storage", request.Host, request.Path)
	// Handle an incoming file that needs saving to FS
	if request.Action == "add" {
		fmt.Printf("Received message [host: %s][path: %s] [fileName: %s] [fileSize: %d]\n",
			request.Host, request.Path, request.FileName, request.FileSize)

		if err := os.MkdirAll(basePath, 0755); err != nil {
			return err
		}
		if request.IsDir {
			return os.MkdirAll(filepath.Join(basePath, request.FileName), 0755)
		}
		return ioutil.WriteFile(filepath.Join(basePath, request.FileName), []byte(strconv.Itoa(request.FileSize)), 0755)
	}
	// Handle a report request
	if request.Action == "query" {

		c, err := ioutil.ReadDir(basePath)
		check(err)

		message := "REPORT:\n"

		fmt.Printf("Listing %s\n", basePath)
		for _, entry := range c {
			fileSize := []byte("")
			fmt.Println(" ", entry.Name(), string(fileSize))
			if !entry.IsDir() {
				fileSize, _ = ioutil.ReadFile(filepath.Join(basePath, entry.Name()))
			} else {
				dirSize := entry.Size()
				fileSizeTmp := scanSubDir(filepath.Join(basePath, entry.Name()))
				fileSizeInt, _ := strconv.ParseInt(string(fileSizeTmp), 10, 64)
				// We need to add the directory size to the total file size
				dirSize += fileSizeInt
				fileSize = []byte(strconv.FormatInt(dirSize, 10)) //int64 to string in base 10
			}

			message += fmt.Sprintf(" %s %s bytes\n", entry.Name(), string(fileSize))
		}

		response, _ := json.Marshal(persistorResponseDTO{
			Message: message,
		})
		return send(conn, []byte(response))
	}
	return nil
}

func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: handle graceful quit
	for {
		func() {
			// Waits on new connections
			conn, err := (*pListener).Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer conn.Close()

			// Decodes new incoming message
			var persistorRequest persistorRequestDTO
			if err = json.NewDecoder(conn).Decode(&persistorRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [path: %s] [action: %s] [file: %s %d] [dir: %v]\n",
				id, persistorRequest.Host, persistorRequest.Path, persistorRequest.Action,
				persistorRequest.FileName, persistorRequest.FileSize, persistorRequest.IsDir)

			if err := handleRequest(&conn, persistorRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	var wg sync.WaitGroup

	fmt.Println("Listening on " + setup.getPersistorHost())

	listener, err = net.Listen("tcp", setup.getPersistorHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= setup.getWorkersPersistor(); w++ {
		wg.Add(1)
		go worker(w, &listener, &wg)
	}

	wg.Wait()
}
