package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"
)

var config Setup

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	var err error
	var jsonConfig *os.File
	var byteValue []byte

	jsonConfig, err = os.Open("config.json")
	check(err)
	defer jsonConfig.Close()

	byteValue, err = ioutil.ReadAll(jsonConfig)
	check(err)

	err = json.Unmarshal(byteValue, &config)
	check(err)
}

func getStatus(key string) bool {
	time.Sleep(2 * time.Second)
	return false
}

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

	var err error
	var conn net.Conn
	var statusRequest StatusDTO
	var statusResponse []byte
	var hasFinishedProcessing bool

	conn, err = (*pListener).Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
		return
	}
	defer conn.Close()

	for {
		func() {
			err = json.NewDecoder(conn).Decode(&statusRequest)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [path: %s]\n",
				id, statusRequest.Host, statusRequest.Path)

			hasFinishedProcessing = getStatus(string(statusRequest.Host + ":" + statusRequest.Path))

			if !hasFinishedProcessing {
				statusResponse, err = json.Marshal(
					StatusDTO{
						Host:   statusRequest.Host,
						Path:   statusRequest.Path,
						Status: "PENDING",
						Data:   nil})
				if err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
					return
				}

				if err = send(&conn, statusResponse); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
					return
				}
				return
			}

		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	var workersWg sync.WaitGroup

	fmt.Println("Listening on " + config.StatusURL)

	listener, err = net.Listen("tcp", config.StatusURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= config.WorkersStatus; w++ {
		workersWg.Add(1)
		go worker(w, &listener, &workersWg)
	}

	workersWg.Wait()
}
