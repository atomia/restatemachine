package main

import (
	"github.com/boltdb/bolt"
	"github.com/BurntSushi/toml"
	"net/http"
	"fmt"
	"os"
)

type Config struct {
	Username string
	Password string
	ListenOn string
	TLSCertificateFile string
	TLSKeyFile string
	StateMachinePath string
	DatabasePath string
}

var globalVersionNumber string
var globalConfig Config

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("restatemachine %s\n", globalVersionNumber)
		os.Exit(0)
	}

	if _, err := toml.DecodeFile("/etc/restatemachine/restatemachine.conf", &globalConfig); err != nil {
		// without or with invalid config file, we use defaults
		globalConfig.Username = ""
		globalConfig.ListenOn = ":80"
		globalConfig.StateMachinePath = "/etc/restatemachine/statemachines"
		globalConfig.DatabasePath = "/etc/restatemachine/state.db"
		globalConfig.TLSCertificateFile = ""
	} 

	if globalConfig.StateMachinePath == "" {
		globalConfig.StateMachinePath = "/etc/restatemachine/statemachines"
	}

	if globalConfig.DatabasePath == "" {
		globalConfig.DatabasePath = "/etc/restatemachine/state.db"
	}

	db, dbErr := bolt.Open(globalConfig.DatabasePath, 0600, nil)
	if dbErr != nil {
		fmt.Printf("error opening database %s: %s\n", globalConfig.DatabasePath, dbErr)
	}

	defer db.Close()

	timerQuitChannel := globalScheduler.Init(db)
	defer close(timerQuitChannel)

	initMachines()
	initApi()

	var listenErr error
	if globalConfig.TLSCertificateFile != "" && globalConfig.TLSKeyFile != "" {
		listenErr = http.ListenAndServeTLS(globalConfig.ListenOn, globalConfig.TLSCertificateFile, globalConfig.TLSKeyFile, nil)
	} else {
		listenErr = http.ListenAndServe(globalConfig.ListenOn, nil)
	}

	if listenErr != nil {
		fmt.Printf("error listening on %s: %s\n", globalConfig.ListenOn, listenErr)
		os.Exit(1)
	}
}
