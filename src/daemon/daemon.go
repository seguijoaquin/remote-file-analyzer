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

func buildTraceMessage(host string) string {
	return "Follow your progress with --host " + host + "--status"
}

func startNewAnalysis(id int, launcherRequest launcherRequestDTO) error {
	fmt.Printf("[worker: %d] No previus analysis\n", id)
	// Connect to FTP
	fmt.Printf("[worker: %d] Connecting to ftp://%s\n", id, launcherRequest.Host)
	// Launch new analysis
	fmt.Printf("[worker: %d] Launching new job to FTP processor for %s\n", id, launcherRequest.Host)

	return nil
}

func returnAnalysisReport(id int, launcherConnection *net.Conn, statusResponse statusResponseDTO) error {
	if err := send(launcherConnection, []byte("Status: "+statusResponse.Status+"\n")); err != nil {
		return err
	}
	if statusResponse.Status != statusNotFound {
		// TODO: Query to persistor & return
		if err := send(launcherConnection, []byte("Persistor response:\nfile_1\t\t500\n")); err != nil {
			return err
		}
	}
	return nil
}

func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	// Connecto to StatusController
	statusConn, err := net.Dial("tcp", setup.getStatusHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
		return
	}
	defer statusConn.Close()

	statusDecoder := json.NewDecoder(statusConn)

	// TODO: handle graceful quit
	for {
		func() {
			// Waits on new connections from launcher
			launcherConnection, err := (*pListener).Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer launcherConnection.Close()

			// Decodes new incoming launcher message
			var launcherRequest launcherRequestDTO
			if err = json.NewDecoder(launcherConnection).Decode(&launcherRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [path: %s] [report_status: %v]\n",
				id, launcherRequest.Host, launcherRequest.Path, launcherRequest.Report)

			// Builds Status Controller Message
			action := "add"
			if launcherRequest.Report {
				action = "query"
			}
			statusMessage, err := json.Marshal(
				statusRequestDTO{
					Host:   launcherRequest.Host,
					Path:   launcherRequest.Path,
					Action: action})
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			// Sends query message to Status Controller
			if err = send(&statusConn, statusMessage); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			// Collect Status Controller Response
			var statusResponse statusResponseDTO
			if err = statusDecoder.Decode(&statusResponse); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}

			// If we just wanted a report status of prev analysis
			if launcherRequest.Report {
				// We've sent a "query" to status controller
				// Tell launcher if job is done or pending
				if err := returnAnalysisReport(id, &launcherConnection, statusResponse); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
					return
				}
				return // End loop
			}

			// We've sent an "add" to status controller
			// Only if no previuos analysis exists we launch new job
			if statusResponse.Status == statusNoPrevAnalysis {
				if err := startNewAnalysis(id, launcherRequest); err != nil {
					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
					return
				}
			}

			// Return trace analysis message
			if err = send(&launcherConnection, []byte(buildTraceMessage(launcherRequest.Host))); err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
				return
			}
		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	var wg sync.WaitGroup

	fmt.Println("Listening on " + setup.getDaemonHost())

	listener, err = net.Listen("tcp", setup.getDaemonHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	for w := 1; w <= setup.getWorkersDaemon(); w++ {
		wg.Add(1)
		go worker(w, &listener, &wg)
	}

	wg.Wait()
}
