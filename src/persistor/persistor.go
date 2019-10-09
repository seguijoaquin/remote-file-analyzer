package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

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

			fmt.Printf("[worker: %d] [host: %s] [path: %s] [%s] [%s %d] [dir: %v]\n",
				id, persistorRequest.Host, persistorRequest.Path, persistorRequest.Action,
				persistorRequest.FileName, persistorRequest.FileSize, persistorRequest.IsDir)

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
