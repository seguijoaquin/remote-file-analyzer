package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Configuration holds the info about the package setup
type Configuration struct {
	WorkersDaemon int    `json:"workers_daemon"`
	DaemonURL     string `json:"daemon_url"`
	DaemonPort    string `json:"daemon_port"`
	StatusURL     string `json:"status_url"`
	ProcessorURL  string `json:"processor_url"`
}

// setup holds the configuration parameters to all project structure
// Host URLs and number of workers from every app of the project
// Var() is called before init() and init() is called before another package
// imports this package, so this way we guarantee that by the time
// another package imports this, the setup configuration is already initialized
//
// ref: https://golang.org/doc/effective_go.html#init
var (
	setup Configuration
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	var err error
	var jsonConfig *os.File
	var byteValue []byte

	jsonConfig, err = os.Open("config.json")
	check(err)
	defer jsonConfig.Close()

	byteValue, err = ioutil.ReadAll(jsonConfig)
	check(err)

	err = json.Unmarshal(byteValue, &setup)
	check(err)
}

// GetDaemonHost is responsible for the Daemon component URL + Port in config file
func GetDaemonHost() string {
	return setup.DaemonURL + ":" + setup.DaemonPort
}

// GetStatusURL returns the Status component URL in config file
func GetStatusURL() string {
	return setup.StatusURL
}

// GetProcessorURL returns the Processor component URL in config file
func GetProcessorURL() string {
	return setup.ProcessorURL
}

// GetWorkersDaemon returns number of workers assigned to Daemon component in config file
func GetWorkersDaemon() int {
	return setup.WorkersDaemon
}
