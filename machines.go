package main

import (
	"io/ioutil"
	"fmt"
	"os"
	"path"
)

type StateMachine struct {
	Name string
	Path string
	Usage string
	States []string
}

var globalStateMachines []StateMachine

func initMachines() {
	machines, err := ioutil.ReadDir(globalConfig.StateMachinePath)
	if err != nil {
		fmt.Printf("error listing StateMachinePath %s directory: %s\n", globalConfig.StateMachinePath, err)
		os.Exit(1)
	}

	for _, machine := range machines {
	
		machinePath := path.Join(globalConfig.StateMachinePath, machine.Name())

		machineInfo, err := os.Stat(machinePath)
		if err != nil {
			fmt.Printf("error stat'ing %s: %s\n", machinePath, err)
		}

		if machineInfo.IsDir() {
			machineStruct := StateMachine{Name: machine.Name(), Path: machinePath}

			states, err := ioutil.ReadDir(machinePath)
			if err != nil {
				fmt.Printf("error listing %s directory: %s\n", machine.Name(), err)
				os.Exit(1)
			}

			hasStart := false
			for _, stateInfo := range states {
				if stateInfo.Mode().Perm() & 0111 > 0 {
					machineStruct.States = append(machineStruct.States, stateInfo.Name())

					if stateInfo.Name() == "start" {
						hasStart = true
					}
				}
			}

			if !hasStart {
				fmt.Printf("state machine directory for %s has no start state\n", machineStruct.Name)
				os.Exit(1)
			}

			machineStruct.Usage = "TODO"

			globalStateMachines = append(globalStateMachines, machineStruct)
		}
	}
}
