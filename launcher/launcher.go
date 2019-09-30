package launcher

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
)

// LauncherDTO holds the info of
type LauncherDTO struct {
	Status bool   `json:"status"`
	Host   string `json:"host"`
	Path   string `json:"path"`
	Data   string `json:"data"`
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}

func handleArguments() LauncherDTO {
	pHost := flag.String("host", "", "The IP address of the host we want to analyze")
	pStatus := flag.Bool("status", false, "Returns the host analisys status (needs a host)")
	pPath := flag.String("path", "/", "The root path of the analysis")
	flag.Parse()

	if *pHost == "" {
		fmt.Println("Error: No host specified")
		fmt.Println("Use -h or --help flags to get help.")
		os.Exit(1)
	}

	return LauncherDTO{Host: *pHost, Status: *pStatus, Path: *pPath}
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

func main() {
	var message LauncherDTO
	var conn net.Conn
	var err error
	var scanner *bufio.Scanner
	var request []byte

	message = handleArguments()

	conn, err = net.Dial("tcp", ":8081")
	handleError(err)
	defer conn.Close()

	request, err = json.Marshal(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}

	fmt.Println("Request: " + string(request))

	if err = send(&conn, request); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}

	fmt.Println("Response:")

	scanner = bufio.NewScanner(conn)

	for scanner.Scan() {
		fmt.Print(scanner.Text() + "\n")
	}

	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
	}
}
