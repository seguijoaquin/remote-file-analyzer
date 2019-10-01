package launcher

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/seguijoaquin/remote-file-analyzer/src/config"
	"github.com/seguijoaquin/remote-file-analyzer/src/network"
)

// RequestDTO holds the info of
type RequestDTO struct {
	Status bool   `json:"status"`
	Host   string `json:"host"`
	Path   string `json:"path"`
	Data   string `json:"data"`
}

// ResponseDTO holds the info of
type ResponseDTO struct {
	Message string `json:"message"`
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}

func handleArguments() RequestDTO {
	pHost := flag.String("host", "", "The IP address of the host we want to analyze")
	pStatus := flag.Bool("status", false, "Returns the host analisys status (needs a host)")
	pPath := flag.String("path", "/", "The root path of the analysis")
	flag.Parse()

	if *pHost == "" {
		fmt.Println("Error: No host specified")
		fmt.Println("Use -h or --help flags to get help.")
		os.Exit(1)
	}

	return RequestDTO{Host: *pHost, Status: *pStatus, Path: *pPath}
}

func main() {
	message := handleArguments()
	conn, err := net.Dial("tcp", config.GetDaemonHost())
	handleError(err)
	defer conn.Close()

	request, err := json.Marshal(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return // We return gracefully to call deferred methods
	}

	fmt.Println("Request: " + string(request))

	if err := network.Send(&conn, request); err != nil {
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
