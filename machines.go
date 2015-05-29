package main

import (
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
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

			cmd := exec.Command(machinePath + "/start", "--help")
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("error executing '%s/start --help' to get usage for %s: %s\n", machinePath, machineStruct.Name, err)
				os.Exit(1)
			} else {
				machineStruct.Usage = string(output)
			}

			globalStateMachines = append(globalStateMachines, machineStruct)
		}
	}
}

func machineGet(name string) *StateMachine {
        for _, machine := range globalStateMachines {
                if machine.Name == name {
                        return &machine
                }
        }

	return nil
}


type ExecuteResponse struct {
	Id uint64
	Message string
}

func machineExecute(name string, input string) (int, string, *ExecuteResponse) {
	machine := machineGet(name)
	if machine == nil {
		return 404, "State machine not found", nil
	}

	id, err := globalScheduler.ScheduleMachine(name, machine.Path, input)
	if err != nil {
		return 500, fmt.Sprintf("Error scheduling execution of %s: %s", name, err), nil
	} else {
		return -1, "", &ExecuteResponse{Id: id, Message: fmt.Sprintf("The state machine %s was scheduled for execution successfully.")}
	}
}
