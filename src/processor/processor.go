package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
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

func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	// Local cache connection
	//ftpCache := make(map[string]net.Conn)

	for {
		func() {
			// Waits on connections from Daemon or FTP Processor
			conn, err := (*pListener).Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer conn.Close()

			// Decodes new incoming message
			var processorRequest processorRequestDTO
			if err = json.NewDecoder(conn).Decode(&processorRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [path: %s] [user: %v]\n",
				id, processorRequest.Host, processorRequest.Path, processorRequest.User)

			// TODO: connect to FTP & login
			ftpConnection, err := net.Dial("tcp", processorRequest.Host+":21")
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer ftpConnection.Close()

			fmt.Printf("[worker: %d] connected to %s\n", id, setup.getStatusHost())

			processorMessage, err := json.Marshal(
				processorResponseDTO{
					Message: "Login OK",
					Error:   ""})
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			// Sends query message to Status Controller
			if err := send(&conn, processorMessage); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	var workersWg sync.WaitGroup

	fmt.Println("Listening on " + setup.getProcessorHost())

	listener, err = net.Listen("tcp", setup.getProcessorHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= setup.getWorkersProcessor(); w++ {
		workersWg.Add(1)
		go worker(w, &listener, &workersWg)
	}

	workersWg.Wait()
}
