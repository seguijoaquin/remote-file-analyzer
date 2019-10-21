package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type ftpMetadata struct {
	fileName string
	fileSize string
	isDir    bool
}

type connectionsPool struct {
	container map[string]*net.Conn
	mu        *sync.Mutex
}

type ftpWorker struct {
	id        int
	pListener *net.Listener
	wg        *sync.WaitGroup
	ftpConns  *connectionsPool
}

func newConnectionsPool() *connectionsPool {
	return &connectionsPool{
		container: make(map[string]*net.Conn),
		mu:        &sync.Mutex{},
	}
}

func newFtpWorker(id int, pListener *net.Listener, wg *sync.WaitGroup, ftpConns *connectionsPool) *ftpWorker {
	return &ftpWorker{
		id:        id,
		pListener: pListener,
		wg:        wg,
		ftpConns:  ftpConns,
	}
}

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
		fmt.Printf("[worker: %d] [CLIENT] Found connection in container Pool for: %s\n", w.id, id)
		return w.ftpConns.container[id], nil
	}

	w.ftpConns.mu.Lock()
	defer w.ftpConns.mu.Unlock()
	conn, err = w.login(processorRequest)
	if err != nil {
		return nil, err
	}
	w.ftpConns.container[id] = conn

	return conn, nil
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

func (w *ftpWorker) listSubdir(request processorRequestDTO, fileName string) error {
	processorConn, err := net.Dial("tcp", setup.getProcessorHost())
	if err != nil {
		return err
	}
	defer processorConn.Close()

	processorRequest, err := json.Marshal(processorRequestDTO{
		Host:   request.Host,
		Path:   filepath.Join(request.Path, fileName),
		Action: "LIST",
	})
	if err != nil {
		return err
	}

	return send(w.id, &processorConn, processorRequest)
}

func (w *ftpWorker) handleRunError(err error, conn *net.Conn) bool {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[worker: %d] [CLIENT] Fatal error: %s\n", w.id, err.Error())
		if err := w.reply(conn, "", err.Error()); err != nil {
			fmt.Fprintf(os.Stderr, "[worker: %d] [CLIENT] Fatal error: %s\n", w.id, err.Error())
		}
		return true
	}
	return false
}

func (w *ftpWorker) run() {
	defer w.wg.Done()

	for {
		func() {
			// Waits on connections from Daemon or FTP Processor
			processorConn, err := (*w.pListener).Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
				return
			}
			defer processorConn.Close()

			// Decodes new incoming message from Daemon or FTP Processor
			var processorRequest processorRequestDTO
			if err = json.NewDecoder(processorConn).Decode(&processorRequest); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
				return
			}

			fmt.Printf("[worker: %d] [host: %s] [user: %v] [path: %s]\n",
				w.id, processorRequest.Host, processorRequest.User, processorRequest.Path)

			// We get the FTP server connection and it's reader
			ftpConn, err := w.getConnection(processorRequest)
			if w.handleRunError(err, &processorConn) {
				return
			}
			connReader := bufio.NewReader(*ftpConn)

			// If its just a login, we return early
			if processorRequest.Action == "LOGIN" {
				if err := w.reply(&processorConn, "OK", ""); err != nil {
					w.handleRunError(err, &processorConn)
				}
				return
			}

			// Send PASV command and check response
			w.sendCommand(ftpConn, []byte("PASV\n"))
			message, err := connReader.ReadString('\n')
			if w.handleRunError(err, &processorConn) {
				return
			}
			err = w.checkResponse(message, "227")
			if w.handleRunError(err, &processorConn) {
				return
			}

			// Scan PASV response and build HOST:PORT to FTP Data Conn
			var portPref, portSuff, port, ip1, ip2, ip3, ip4 int
			_, err = fmt.Sscanf(message, "227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).",
				&ip1, &ip2, &ip3, &ip4, &portPref, &portSuff)
			if w.handleRunError(err, &processorConn) {
				return
			}
			port = portPref*256 + portSuff
			dataHost := processorRequest.Host + ":" + fmt.Sprintf("%d", port)

			// Make FTP Data Conn
			dataConn, err := net.Dial("tcp", dataHost)
			if w.handleRunError(err, &processorConn) {
				return
			}
			defer dataConn.Close()

			// Send LIST command and check response
			w.sendCommand(ftpConn, []byte(fmt.Sprintf("LIST %s\n", processorRequest.Path)))
			message, err = connReader.ReadString('\n')
			if w.handleRunError(err, &processorConn) {
				return
			}
			err = w.checkResponse(message, "150")
			if w.handleRunError(err, &processorConn) {
				return
			}

			// Check for finished Listing
			message, err = connReader.ReadString('\n')
			if w.handleRunError(err, &processorConn) {
				return
			}
			err = w.checkResponse(message, "226")
			if w.handleRunError(err, &processorConn) {
				return
			}

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
				//fmt.Printf("[worker: %d] %s\n", w.id, buff)
			}
			if err := dconnbuf.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
				return
			}

			for _, val := range files {
				if err := w.store(processorRequest, val.fileName, val.fileSize, val.isDir); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
					break
				}
				if err := w.updateStatus(processorRequest, val.fileName, val.isDir); err != nil {
					fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
					break
				}
				if val.isDir {
					if err := w.listSubdir(processorRequest, val.fileName); err != nil {
						fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
						break
					}
				}
			}

			// Tell Daemon or Processor it's OK
			if err := w.reply(&processorConn, "OK", ""); err != nil {
				fmt.Fprintf(os.Stderr, "[worker: %d] Fatal error: %s\n", w.id, err.Error())
			}
		}()
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

	for w := 1; w <= setup.getWorkersProcessor(); w++ {
		workersWg.Add(1)
		worker := newFtpWorker(w, &listener, &workersWg, ftpConnections)
		go worker.run()
	}

	workersWg.Wait()
}
