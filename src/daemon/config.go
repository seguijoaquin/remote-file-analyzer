package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Configuration holds the info about the package setup
type Configuration struct {
	WorkersDaemon    int    `json:"workers_daemon"`
	WorkersPersistor int    `json:"workers_persistor"`
	WorkersProcessor int    `json:"workers_processor"`
	DaemonURL        string `json:"daemon_url"`
	DaemonPort       string `json:"daemon_port"`
	StatusURL        string `json:"status_url"`
	StatusPort       string `json:"status_port"`
	ProcessorURL     string `json:"processor_url"`
	ProcessorPort    string `json:"processor_port"`
	PersistorURL     string `json:"persistor_url"`
	PersistorPort    string `json:"persistor_port"`
}

// setup holds the configuration parameters to all project structure
// Host URLs and number of workers from every app of the project
// Var() is called before init() and init() is called before another package
// imports this package, so this way we guarantee that by the time
// another package imports this, the setup configuration is already initialized
//
// ref: https://golang.org/doc/effective_go.html#init
var (
	setup                Configuration
	statusNotFound       = "NOT_FOUND"
	statusNoPrevAnalysis = "NO_PREV_ANALYSIS"
	statusAlreadyExists  = "ANALYSIS_ALREADY_EXISTS"
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

func (s Configuration) getDaemonHost() string {
	return s.DaemonURL + ":" + s.DaemonPort
}

func (s Configuration) getStatusHost() string {
	return s.StatusURL + ":" + s.StatusPort
}

func (s Configuration) getPersistorHost() string {
	return s.PersistorURL + ":" + s.PersistorPort
}

func (s Configuration) getProcessorHost() string {
	return s.ProcessorURL + ":" + s.ProcessorPort
}

func (s Configuration) getWorkersDaemon() int {
	return s.WorkersDaemon
}

func (s Configuration) getWorkersPersistor() int {
	return s.WorkersPersistor
}

func (s Configuration) getWorkersProcessor() int {
	return s.WorkersProcessor
}
