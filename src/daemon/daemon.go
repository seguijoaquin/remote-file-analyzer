package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"sync"

	"github.com/seguijoaquin/remote-file-analyzer/src/network"

	"github.com/seguijoaquin/remote-file-analyzer/src/launcher"

	"github.com/seguijoaquin/remote-file-analyzer/src/config"
)

func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	launcherConnection, err := (*pListener).Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
		debug.PrintStack()
		return
	}
	defer launcherConnection.Close()

	var launcherRequest launcher.RequestDTO
	if err = json.NewDecoder(launcherConnection).Decode(&launcherRequest); err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
		return
	}

	network.Send(&launcherConnection, []byte("You said: "+launcherRequest.Host))

}

// func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
// 	defer wg.Done()

// 	var err error
// 	var launcherConn net.Conn
// 	var launcherRequest LauncherDTO
// 	var statusResponse StatusDTO
// 	var statusConn net.Conn
// 	var statusDecoder *json.Decoder

// 	if statusConn, err = net.Dial("tcp", config.StatusURL); err != nil {
// 		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 		debug.PrintStack()
// 		return
// 	}
// 	defer statusConn.Close()

// 	statusDecoder = json.NewDecoder(statusConn)

// 	for {
// 		func() {
// 			if launcherConn, err = (*pListener).Accept(); err != nil {
// 				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 				return
// 			}
// 			defer launcherConn.Close()

// 			if err = json.NewDecoder(launcherConn).Decode(&launcherRequest); err != nil {
// 				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 				return
// 			}

// 			fmt.Printf("[worker: %d] [host: %s] [path: %s] [report_status: %v]\n",
// 				id, launcherRequest.Host, launcherRequest.Path, launcherRequest.Status)

// 			var query []byte
// 			if query, err = json.Marshal(StatusDTO{Host: launcherRequest.Host, Path: launcherRequest.Path}); err != nil {
// 				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 				return
// 			}

// 			if err = send(&statusConn, query); err != nil {
// 				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 				return
// 			}

// 			if err = statusDecoder.Decode(&statusResponse); err != nil {
// 				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
// 				return
// 			}

// 			if statusResponse.Status == "PENDING" {
// 				if err = send(&launcherConn, []byte("PROCESSING...\n")); err != nil {
// 					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 					return
// 				}
// 			} else {
// 				if err = send(&launcherConn, statusResponse.Data); err != nil {
// 					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
// 					return
// 				}
// 			}
// 		}()
// 	}
// }

func main() {
	var err error
	var listener net.Listener
	var wg sync.WaitGroup

	fmt.Println("Listening on " + config.GetDaemonHost())

	listener, err = net.Listen("tcp", config.GetDaemonHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= config.GetWorkersDaemon(); w++ {
		wg.Add(1)
		go worker(w, &listener, &wg)
	}

	wg.Wait()
}
