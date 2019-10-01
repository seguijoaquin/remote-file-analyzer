// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"net"
// 	"os"
// 	"sync"
// 	"time"
// )

// const ftpTimeout time.Duration = 30 * time.Second

// var config Setup

// func check(e error) {
// 	if e != nil {
// 		panic(e)
// 	}
// }

// func init() {
// 	var err error
// 	var jsonConfig *os.File
// 	var byteValue []byte

// 	jsonConfig, err = os.Open("config.json")
// 	check(err)
// 	defer jsonConfig.Close()

// 	byteValue, err = ioutil.ReadAll(jsonConfig)
// 	check(err)

// 	err = json.Unmarshal(byteValue, &config)
// 	check(err)
// }

// func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
// 	defer wg.Done()

// 	var err error
// 	var conn net.Conn
// 	var daemonRequest daemonDTO

// 	// Local cache connection to ftp clients
// 	var ftpCache map[string]net.Conn

// 	conn, err = (*pListener).Accept()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 		return
// 	}
// 	defer conn.Close()

// 	ftpCache = make(map[string]net.Conn)

// 	for {
// 		func() {
// 			err = json.NewDecoder(conn).Decode(&daemonRequest)
// 			if err != nil {
// 				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 				return
// 			}

// 			cacheKey := daemonRequest.User + ":" + daemonRequest.Host

// 			fmt.Printf("[worker: %d] [host: %s] [path: %s] [user: %s]\n",
// 				id, daemonRequest.Host, daemonRequest.Path, daemonRequest.User)

// 			if ftpConn, ok := ftpCache[cacheKey]; !ok {
// 				ftpConn, err = net.Dial("tcp", daemonRequest.Host+":21")
// 				if err != nil {
// 					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 					return
// 				}
// 				ftpCache[cacheKey] = ftpConn
// 			} else {
// 				// TODO: Check if connection is closed & reopen
// 			}

// 			// Refresh connection
// 			ftpCache[cacheKey].SetDeadline(time.Now().Add(ftpTimeout))

// 			// Write to ftp
// 			// ftpCache[cacheKey].Write([]byte("ls"))
// 			// Read ftp response
// 			// var buff []byte
// 			// ftpCache[cacheKey].Read(&buff)

// 		}()
// 	}
// }

// func main() {
// 	var err error
// 	var listener net.Listener
// 	var workersWg sync.WaitGroup

// 	fmt.Println("Listening on " + config.ProcessorURL)

// 	listener, err = net.Listen("tcp", config.ProcessorURL)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 		return
// 	}
// 	defer listener.Close()

// 	for w := 1; w <= config.WorkersProcessor; w++ {
// 		workersWg.Add(1)
// 		go worker(w, &listener, &workersWg)
// 	}

// 	workersWg.Wait()
// }

package main

import "fmt"

func main() {
	fmt.Println("Hola mundo, soy processor")
}
