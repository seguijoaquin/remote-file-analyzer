package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Send is responsible for sending messages through a
// generic stream-oriented network connection
func send(id int, conn *net.Conn, request []byte) error {
	var err error
	var n int
	n, err = (*conn).Write(request)

	// fmt.Printf("[worker: %d] SENDING to (%s): %s\n", id, (*conn).RemoteAddr(), request)

	for n < len(request) {
		n, err = (*conn).Write(request[n:])
		if err != nil {
			break
		}
	}
	return err
}

type ftpMetadata struct {
	fileName string
	fileSize string
	isDir    bool
}

type ftpConnection struct {
	conn *net.Conn
	mu   *sync.Mutex
}

type connectionsPool struct {
	container map[string]*ftpConnection
	mu        *sync.Mutex
}

type ftpWorker struct {
	id           int
	wg           *sync.WaitGroup
	ftpConns     *connectionsPool
	connsChan    chan net.Conn
	requestsChan chan processorRequestDTO
}

func newConnectionsPool() *connectionsPool {
	return &connectionsPool{
		container: make(map[string]*ftpConnection),
		mu:        &sync.Mutex{},
	}
}

func newFtpWorker(id int, wg *sync.WaitGroup, ftpConns *connectionsPool, connsChan chan net.Conn, reqChan chan processorRequestDTO) *ftpWorker {
	return &ftpWorker{
		id:           id,
		wg:           wg,
		ftpConns:     ftpConns,
		connsChan:    connsChan,
		requestsChan: reqChan,
	}
}

func (w *ftpWorker) sendCommand(conn *net.Conn, request []byte) error {
	var err error
	var n int
	n, err = fmt.Fprintf((*conn), string(request))

	for n < len(request) {
		n, err = fmt.Fprintf((*conn), string(request[n:]))
		if err != nil {
			break
		}
	}
	fmt.Printf("[worker: %d] [CLIENT] %s", w.id, string(request))
	return err
}

func (w *ftpWorker) checkResponse(message string, code string) error {
	fmt.Printf("[worker: %d] [SERVER] %s", w.id, message)
	if !(strings.Contains(message, code)) {
		return fmt.Errorf("Expected: %s but received: %s", code, message)
	}
	return nil
}

func (w *ftpWorker) reply(conn *net.Conn, message string, errMsg string) error {
	processorMessage, err := json.Marshal(
		processorResponseDTO{
			Message: message,
			Error:   errMsg})
	if err != nil {
		return err
	}

	return send(w.id, conn, processorMessage)
}

func (w *ftpWorker) login(processorRequest processorRequestDTO) (*net.Conn, error) {
	host := processorRequest.Host + ":21"
	conn, err := net.Dial("tcp", host) // We won't close this since we will return it
	if err != nil {
		return nil, fmt.Errorf("[CLIENT] Fatal error: %s", err.Error())
	}

	connReader := bufio.NewReader(conn)
	message, _ := connReader.ReadString('\n')
	if err := w.checkResponse(message, "220"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("[CLIENT] Fatal error: %s", err.Error())
	}

	w.sendCommand(&conn, []byte("USER "+processorRequest.User+"\n"))
	message, _ = connReader.ReadString('\n')
	if err := w.checkResponse(message, "331"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("[CLIENT] Fatal error: %s", err.Error())
	}

	w.sendCommand(&conn, []byte("PASS "+processorRequest.Password+"\n"))
	message, _ = connReader.ReadString('\n')
	if err := w.checkResponse(message, "230"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("[CLIENT] Fatal error: %s", err.Error())
	}

	fmt.Printf("[worker: %d] [CLIENT] Logged in to FTP server\n", w.id)

	return &conn, nil
}

func (w *ftpWorker) getConnection(processorRequest processorRequestDTO) (conn *net.Conn, err error) {
	id := processorRequest.Host
	if _, ok := w.ftpConns.container[id]; ok {
		w.ftpConns.container[id].mu.Lock()
		fmt.Printf("[worker: %d] [CLIENT] Found connection in container Pool for: %s\n", w.id, id)
		return w.ftpConns.container[id].conn, nil
	}

	return nil, fmt.Errorf("Connection not found for host: %s", processorRequest.Host)

	// w.ftpConns.mu.Lock()
	// defer w.ftpConns.mu.Unlock()
	// conn, err = w.login(processorRequest)
	// if err != nil {
	// 	return nil, err
	// }
	// w.ftpConns.container[id] = &ftpConnection{conn: conn, mu: &sync.Mutex{}}

	// return conn, nil
}

func (w *ftpWorker) releaseConnection(processorRequest processorRequestDTO) {
	w.ftpConns.container[processorRequest.Host].mu.Unlock()
}

func (w *ftpWorker) saveConnection(conn *net.Conn, processorRequest processorRequestDTO) {
	id := processorRequest.Host
	w.ftpConns.mu.Lock()
	defer w.ftpConns.mu.Unlock()
	w.ftpConns.container[id] = &ftpConnection{conn: conn, mu: &sync.Mutex{}}
}

func (w *ftpWorker) store(request processorRequestDTO, fileName string, sFileSize string, isDir bool) error {
	persistorConn, err := net.Dial("tcp", setup.getPersistorHost())
	if err != nil {
		return err
	}
	defer persistorConn.Close()

	fileSize, _ := strconv.Atoi(sFileSize)

	persistorRequest, err := json.Marshal(persistorRequestDTO{
		Host:     request.Host,
		Action:   "add",
		Path:     request.Path,
		FileName: fileName,
		FileSize: fileSize,
		IsDir:    isDir,
	})
	if err != nil {
		return err
	}

	return send(w.id, &persistorConn, persistorRequest)
}

func (w *ftpWorker) updateStatus(request processorRequestDTO, fileName string, isDir bool) error {
	statusConn, err := net.Dial("tcp", setup.getStatusHost())
	if err != nil {
		return err
	}
	defer statusConn.Close()

	// If it is a directory, we add one to the pending status count
	// If it is a file, we substract one to the pending status count
	pending := -1
	if isDir {
		pending = 1
	}

	statusRequest, err := json.Marshal(statusRequestDTO{
		Host:    request.Host,
		Action:  "update",
		Path:    request.Path,
		Pending: pending,
	})
	if err != nil {
		return err
	}

	return send(w.id, &statusConn, statusRequest)
}

func (w *ftpWorker) listSubdir(request processorRequestDTO, fileName string) {
	w.requestsChan <- processorRequestDTO{
		Host:   request.Host,
		Path:   filepath.Join(request.Path, fileName),
		Action: "LIST",
	}
}

// func (w *ftpWorker) handleRunError(err error, conn *net.Conn) bool {
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "[worker: %d] [CLIENT] Fatal error: %s\n", w.id, err.Error())
// 		if err := w.reply(conn, "", err.Error()); err != nil {
// 			fmt.Fprintf(os.Stderr, "[worker: %d] [CLIENT] Fatal error: %s\n", w.id, err.Error())
// 		}
// 		return true
// 	}
// 	return false
// }

func (w *ftpWorker) failOnError(err error) {
	if err != nil {
		log.Fatalf("[worker: %d] Fatal error: %s\n", w.id, err.Error())
	}
}

func (w *ftpWorker) processRequest(processorRequest processorRequestDTO) {
	// Send PASV command and check response
	ftpConn, err := w.getConnection(processorRequest)
	w.failOnError(err)
	connReader := bufio.NewReader(*ftpConn)
	w.failOnError(w.sendCommand(ftpConn, []byte("PASV\n")))

	message, err := connReader.ReadString('\n')
	w.failOnError(err)
	w.failOnError(w.checkResponse(message, "227"))

	// Scan PASV response and build HOST:PORT to FTP Data Conn
	var portPref, portSuff, port, ip1, ip2, ip3, ip4 int
	_, err = fmt.Sscanf(message, "227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).",
		&ip1, &ip2, &ip3, &ip4, &portPref, &portSuff)
	w.failOnError(err)
	port = portPref*256 + portSuff
	dataHost := processorRequest.Host + ":" + fmt.Sprintf("%d", port)

	// Make FTP Data Conn
	dataConn, err := net.Dial("tcp", dataHost)
	w.failOnError(err)

	// Send LIST command and check response
	w.failOnError(w.sendCommand(ftpConn, []byte(fmt.Sprintf("LIST %s\n", processorRequest.Path))))
	message, err = connReader.ReadString('\n')
	w.failOnError(err)
	w.failOnError(w.checkResponse(message, "150"))

	// Check for finished Listing
	message, err = connReader.ReadString('\n')
	w.failOnError(err)
	w.failOnError(w.checkResponse(message, "226"))

	w.releaseConnection(processorRequest)

	dconnbuf := bufio.NewScanner(dataConn)

	var files []ftpMetadata

	// Parse response from FTP Data Conn and
	// send response to Status Controller, Persistor and Storage
	for dconnbuf.Scan() {
		buff := dconnbuf.Text()
		fields := strings.Fields(buff)
		files = append(files, ftpMetadata{
			fileName: fields[8],
			fileSize: fields[4],
			isDir:    strings.Contains(fields[0], "d"),
		})
	}

	w.failOnError(dconnbuf.Err())

	for _, val := range files {
		if err := w.store(processorRequest, val.fileName, val.fileSize, val.isDir); err != nil {
			fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
			break
		}
		if err := w.updateStatus(processorRequest, val.fileName, val.isDir); err != nil {
			fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
			break
		}
		if val.isDir && val.fileName != "sys" {
			w.listSubdir(processorRequest, val.fileName)
		}
	}
}

func (w *ftpWorker) run() {
	defer w.wg.Done()

	for {
		select {
		case processorConn := <-w.connsChan:

			// Decodes new incoming connection from channel
			var processorRequest processorRequestDTO
			w.failOnError(json.NewDecoder(processorConn).Decode(&processorRequest))

			fmt.Printf("[worker: %d] [host: %s] [user: %v] [path: %s]\n",
				w.id, processorRequest.Host, processorRequest.User, processorRequest.Path)

			// If its just a login, we return early
			if processorRequest.Action == "LOGIN" {
				ftpConn, _ := w.login(processorRequest)
				w.saveConnection(ftpConn, processorRequest)
				w.reply(&processorConn, "OK", "")
				return
			}

			w.requestsChan <- processorRequest

			w.reply(&processorConn, "OK", "")

			processorConn.Close()
		case request := <-w.requestsChan:
			w.processRequest(request)
		}
	}
}

func main() {
	var err error
	var listener net.Listener
	var workersWg sync.WaitGroup

	fmt.Println("Listening on " + setup.getProcessorHost())

	listener, err = net.Listen("tcp", setup.getProcessorHost())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return
	}
	defer listener.Close()

	ftpConnections := newConnectionsPool()
	processorConnectionsChan := make(chan net.Conn, 1000)
	processorRequestsChan := make(chan processorRequestDTO, 10000)

	for w := 1; w <= setup.getWorkersProcessor(); w++ {
		workersWg.Add(1)
		processorWorker := newFtpWorker(w, &workersWg, ftpConnections, processorConnectionsChan, processorRequestsChan)
		go processorWorker.run()
	}

	for {
		processorConn, err := (listener).Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
			continue
		}
		processorConnectionsChan <- processorConn
	}

	workersWg.Wait()
}
