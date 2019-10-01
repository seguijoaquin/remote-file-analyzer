package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

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

func handleQueryAction(pendingFiles map[string]int, statusRequest statusRequestDTO, conn *net.Conn) error {

	status := statusNotFound
	pending := 0
	if value, ok := pendingFiles[statusRequest.Host]; ok {
		if value == 0 {
			status = "FINALIZADO"
		} else {
			status = "PROCESANDO"
			pending = pendingFiles[statusRequest.Host]
		}
	}

	statusResponse, err := json.Marshal(
		statusResponseDTO{
			Status:  status,
			Pending: pending})
	if err != nil {
		return err
	}

	return send(conn, statusResponse)
}

func handleAddAction(pendingFiles map[string]int, statusRequest statusRequestDTO, conn *net.Conn) error {
	status := statusNoPrevAnalysis
	if _, ok := pendingFiles[statusRequest.Host]; ok {
		status = statusAlreadyExists
		if statusRequest.Action == "update" {
			pendingFiles[statusRequest.Host] += statusRequest.Pending
		}
	} else {
		// If it does not exist we add it
		pendingFiles[statusRequest.Host] = 0
	}

	statusMessage, err := json.Marshal(statusResponseDTO{
		Status: status, Pending: pendingFiles[statusRequest.Host]})
	if err != nil {
		return err
	}

	return send(conn, statusMessage)
}

func main() {
	fmt.Println("Listening on " + setup.getStatusHost())

	listener, err := net.Listen("tcp", setup.getStatusHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	// TODO: refresh list with TTL
	pendingFiles := make(map[string]int)

	for {
		func() {
			// Wait for incoming connection from daemon or FTP Processor
			conn, err := listener.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
				return
			}
			defer conn.Close()

			// Receive and decode message
			var statusRequest statusRequestDTO
			if err := json.NewDecoder(conn).Decode(&statusRequest); err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
				return
			}

			if statusRequest.Action == "query" {
				if err := handleQueryAction(pendingFiles, statusRequest, &conn); err != nil {
					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
					return
				}
			} else {
				// Handle action "update" from FTP Processor
				// and action "add" from Daemon
				if err := handleAddAction(pendingFiles, statusRequest, &conn); err != nil {
					fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
					return
				}
			}
		}()
	}
}
