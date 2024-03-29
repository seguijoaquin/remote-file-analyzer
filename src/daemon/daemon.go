package main

import (
	"encoding/json"
	"errors"
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
	return "Follow your progress with --host " + host + " -report"
}

func processorConnect(launcherRequest launcherRequestDTO) error {
	// Connect to to FTP Processor
	processorConn, err := net.Dial("tcp", setup.getProcessorHost())
	if err != nil {
		return err
	}
	defer processorConn.Close()

	processorDecoder := json.NewDecoder(processorConn)

	loginMessage, err := json.Marshal(
		processorRequestDTO{
			Host:     launcherRequest.Host,
			User:     launcherRequest.User,
			Password: launcherRequest.Password,
			Action:   "LOGIN",
		})
	if err != nil {
		return err
	}

	// Sends login to FTP
	if err := send(&processorConn, loginMessage); err != nil {
		return err
	}

	// Collect Processor Response
	var processorResponse processorResponseDTO
	if err := processorDecoder.Decode(&processorResponse); err != nil {
		return err
	}

	if processorResponse.Error != "" {
		return errors.New(processorResponse.Error)
	}
	return nil
}

func launchAnalysis(id int, launcherRequest launcherRequestDTO) error {
	// Connect to to FTP Processor
	processorConn, err := net.Dial("tcp", setup.getProcessorHost())
	if err != nil {
		return err
	}
	defer processorConn.Close()

	processorDecoder := json.NewDecoder(processorConn)

	analysisMessage, err := json.Marshal(
		processorRequestDTO{
			Host:   launcherRequest.Host,
			Path:   launcherRequest.Path,
			Action: "LIST",
		})
	if err != nil {
		return err
	}

	if err := send(&processorConn, analysisMessage); err != nil {
		return err
	}

	// Collect Processor Response
	var processorResponse processorResponseDTO
	if err := processorDecoder.Decode(&processorResponse); err != nil {
		return err
	}

	if processorResponse.Error != "" {
		return errors.New(processorResponse.Error)
	}
	return nil
}

func startNewAnalysis(id int, launcherRequest launcherRequestDTO) error {
	fmt.Printf("[worker: %d] No previus analysis for %s\n", id, launcherRequest.Host)

	// Connect to FTP Processor
	fmt.Printf("[worker: %d] Connecting to ftp://%s\n", id, launcherRequest.Host)
	if err := processorConnect(launcherRequest); err != nil {
		return err
	}
	// Launch new analysis
	fmt.Printf("[worker: %d] Launching new job to FTP processor for %s\n", id, launcherRequest.Host)

	return launchAnalysis(id, launcherRequest)
}

func returnAnalysisReport(id int, launcherConnection *net.Conn, launcherRequest launcherRequestDTO, statusResponse statusResponseDTO) error {
	if err := send(launcherConnection, []byte("Status: "+statusResponse.Status+"\n")); err != nil {
		return err
	}
	if statusResponse.Status != statusNotFound {
		// TODO: Query to persistor & return
		conn, err := net.Dial("tcp", setup.getPersistorHost())
		if err != nil {
			return err
		}
		defer conn.Close()
		request, _ := json.Marshal(persistorRequestDTO{
			Action: "query",
			Host:   launcherRequest.Host,
			Path:   launcherRequest.Path,
		})
		if err := send(&conn, request); err != nil {
			return err
		}

		persistorDecoder := json.NewDecoder(conn)
		var persistorMessage persistorResponseDTO
		persistorDecoder.Decode(&persistorMessage)

		return send(launcherConnection, []byte(persistorMessage.Message))
	}
	return nil
}

func worker(id int, pListener *net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

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

			// Connecto to StatusController
			statusConn, err := net.Dial("tcp", setup.getStatusHost())
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
				return
			}
			defer statusConn.Close()

			fmt.Printf("[worker: %d] connected to %s\n", id, setup.getStatusHost())

			statusDecoder := json.NewDecoder(statusConn)

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
				if err := returnAnalysisReport(id, &launcherConnection, launcherRequest, statusResponse); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
					return
				}
				return // End loop
			}

			// We've sent an "add" to status controller
			// Only if no previuos analysis exists we launch new job
			if statusResponse.Status == statusNoPrevAnalysis {
				if err := startNewAnalysis(id, launcherRequest); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", id, err.Error())
					return
				}
			} else {
				fmt.Printf("[worker: %d] A previus analysis was found for %s\n", id, launcherRequest.Host)
			}

			// Return trace analysis message
			if err = send(&launcherConnection, []byte(buildTraceMessage(launcherRequest.Host))); err != nil {
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
