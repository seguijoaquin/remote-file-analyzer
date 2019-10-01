package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
)

type launcherRequest struct {
	Report   bool   `json:"report"`
	Host     string `json:"host"`
	Path     string `json:"path"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Send is responsible for sending messages through a
// generic stream-oriented network connection.
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

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}

func handleArguments() (request launcherRequest, daemonEndpoint string) {
	pHost := flag.String("host", "",
		"The IP address of the host we want to analyze")
	pReport := flag.Bool("report", false,
		"Returns the host analisys status (needs a host)")
	pPath := flag.String("path", "/",
		"The root path of the analysis")
	pDaemonEndpoint := flag.String("daemon", "0.0.0.0:8081",
		"The Daemon endpoint of Remote-File-Analyzer app. Default is 0.0.0.0:8081")
	// TODO: Handle user & password
	flag.Parse()

	if *pHost == "" {
		fmt.Println("Error: No host specified")
		fmt.Println("Use -h or --help flags to get help.")
		os.Exit(1)
	}

	return launcherRequest{Host: *pHost, Report: *pReport, Path: *pPath}, (*pDaemonEndpoint)
}

func main() {
	message, daemonEndpoint := handleArguments()
	conn, err := net.Dial("tcp", daemonEndpoint)
	handleError(err)
	defer conn.Close()

	request, err := json.Marshal(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return // We return gracefully to call deferred methods
	}

	fmt.Println("Request: " + string(request))

	if err := send(&conn, request); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return // Return gracefully to call deferred methods
	}

	fmt.Println("Response:")

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		fmt.Print(scanner.Text() + "\n")
	}

	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		// No need to return here
	}
}
