package main

import (
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
}

var globalConfig Config

func main() {
	if _, err := toml.DecodeFile("/etc/restatemachine/restatemachine.conf", &globalConfig); err != nil {
		// without or with invalid config file, we use defaults
		globalConfig.Username = ""
		globalConfig.ListenOn = ":80"
		globalConfig.StateMachinePath = "/etc/restatemachine/statemachines"
		globalConfig.TLSCertificateFile = ""
	} else if globalConfig.StateMachinePath == "" {
		globalConfig.StateMachinePath = "/etc/restatemachine/statemachines"
	}

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
