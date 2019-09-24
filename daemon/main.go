package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime/debug"
	"sync"
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
	var launcherConn net.Conn
	var launcherRequest LauncherDTO
	var statusResponse StatusDTO
	var statusConn net.Conn
	var statusDecoder *json.Decoder

	if statusConn, err = net.Dial("tcp", config.StatusURL); err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
		debug.PrintStack()
		return
	}
	defer statusConn.Close()

	statusDecoder = json.NewDecoder(statusConn)

	for {
		func() {
			if launcherConn, err = (*pListener).Accept(); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer launcherConn.Close()

			if err = json.NewDecoder(launcherConn).Decode(&launcherRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [path: %s] [report_status: %v]\n",
				id, launcherRequest.Host, launcherRequest.Path, launcherRequest.Status)

			var query []byte
			if query, err = json.Marshal(StatusDTO{Host: launcherRequest.Host, Path: launcherRequest.Path}); err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
				return
			}

			if err = send(&statusConn, query); err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
				return
			}

			if err = statusDecoder.Decode(&statusResponse); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			if statusResponse.Status == "PENDING" {
				if err = send(&launcherConn, []byte("PROCESSING...\n")); err != nil {
					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
					return
				}
			} else {
				if err = send(&launcherConn, statusResponse.Data); err != nil {
					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
					return
				}
			}
		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	var wg sync.WaitGroup

	fmt.Println("Listening on " + config.DaemonURL)

	listener, err = net.Listen("tcp", config.DaemonURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= config.WorkersDaemon; w++ {
		wg.Add(1)
		go worker(w, &listener, &wg)
	}

	wg.Wait()
}
