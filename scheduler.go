package main

import (
	"github.com/boltdb/bolt"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RunningMachine struct {
        Id uint64
        Name string
        Path string
        Input string
	LastState string
	NextState string
	StatusMessage string
	RunningStateCode bool
	NextStateRun time.Time
}

type Scheduler struct {
	SchedulerLock *sync.Mutex
	RunningMachines []RunningMachine
	Database *bolt.DB
}

var globalScheduler Scheduler

func (s *Scheduler) GetRunningMachines() *[]RunningMachine {
	s.SchedulerLock.Lock()
	runningMachines := s.RunningMachines
	s.SchedulerLock.Unlock()

	return &runningMachines
}

func (s *Scheduler) GetMachineRun(id string) (*RunningMachine, error) {
	var machine RunningMachine

	returnErr := s.Database.View(func(tx *bolt.Tx) error {
		runsBucket := tx.Bucket([]byte("MachineRuns"))
		if runsBucket == nil {
			return fmt.Errorf("error getting database bucket")
		}

		machineJson := runsBucket.Get([]byte(id))
		if machineJson == nil {
			return fmt.Errorf("no running state machine with id %s found", id)
		}

		err := json.Unmarshal(machineJson, &machine)
		if err != nil {
			return fmt.Errorf("error deserializing machine run from persisted db: %s", err)
		}

		return nil
	})

	if returnErr != nil {
		return nil, returnErr
	} else {
		return &machine, nil
	}
}

func (s *Scheduler) UpdatePersistedMachine(machine *RunningMachine) (id uint64, returnErr error) {
	returnErr = s.Database.Update(func(tx *bolt.Tx) error {
		runningBucket := tx.Bucket([]byte("RunningMachines"))
		runsBucket := tx.Bucket([]byte("MachineRuns"))
		if runningBucket == nil || runsBucket == nil {
			return fmt.Errorf("error getting database bucket")
		}

		if machine.Id == 0 {
			var seqErr error
			id, seqErr = runsBucket.NextSequence()
			if seqErr != nil {
				return fmt.Errorf("error getting next value in MachineRuns sequence: %s", seqErr)
			}
			machine.Id = id
		} else {
			id = machine.Id
		}

		machineJson, err := json.Marshal(machine)
		if err != nil {
			return fmt.Errorf("error serializing machine as json for persisting: %s", err)
		}

		idKey := []byte(fmt.Sprintf("%d", id))
		err = runsBucket.Put(idKey, machineJson)
		if err != nil {
			return fmt.Errorf("error persisting machine run: %s", err)
		}

		err = runningBucket.Put(idKey, nil)
		if err != nil {
			return fmt.Errorf("error persisting machine run: %s", err)
		}

		return nil
	});

	return
}

func (s *Scheduler) ScheduleMachine(name string, path string, input string) (id uint64, returnErr error) {
	machine := RunningMachine{Id: 0, Name: name, Path: path, Input: input, NextState: "start", RunningStateCode: false, NextStateRun: time.Time{}}
	id, returnErr = s.UpdatePersistedMachine(&machine)
	if returnErr == nil {
		s.AddMachine(&machine)
	}

	return
}

func (s *Scheduler) AddMachine(machine *RunningMachine) {
	s.SchedulerLock.Lock()
	s.RunningMachines = append(s.RunningMachines, *machine)
	s.SchedulerLock.Unlock()
}

func (s *Scheduler) CancelMachineRun(id string) error {
	return s.Database.Update(func(tx *bolt.Tx) error {
		s.SchedulerLock.Lock()
	
		machineIdx := -1
		var machine RunningMachine
		var idx int
		for idx, machine = range s.RunningMachines {
			if fmt.Sprintf("%d", machine.Id) == id {
				machineIdx = idx
				break
			}
		}

		if machineIdx == -1 {
			s.SchedulerLock.Unlock()
			return fmt.Errorf("state machine run with id %s is not currently active", id)
		} else {
			// Delete from in-memory representation right away so that we can drop lock
			s.RunningMachines = append(s.RunningMachines[:machineIdx], s.RunningMachines[machineIdx+1:]...)
			s.SchedulerLock.Unlock()
		}
	
		runningBucket := tx.Bucket([]byte("RunningMachines"))
		runsBucket := tx.Bucket([]byte("MachineRuns"))
		if runningBucket == nil || runsBucket == nil {
			return fmt.Errorf("error getting database bucket")
		}


		err := runningBucket.Delete([]byte(id))
		if err != nil {
			return err
		}

		if machine.NextState != "stop" {
			machine.StatusMessage = "State machine run cancelled manually"
			machine.NextState = "stop"
			machine.RunningStateCode = false

			machineJson, jsonErr := json.Marshal(machine)
			if jsonErr!= nil {
				return fmt.Errorf("error serializing machine as json for persisting: %s", err)
			}

			runsBucket.Put([]byte(id), machineJson)
		}

		return nil
	})
}

func (s *Scheduler) ExecuteState(machine *RunningMachine) {
	cmdPath := machine.Path + "/" + machine.NextState
	cmd := exec.Command(cmdPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(machine.Input)
	err := cmd.Run()

	if err != nil {
		machine.StatusMessage = fmt.Sprintf("error executing state code at %s (will keep retrying): %s", cmdPath, err)
	} else {
		stderrStr := string(stderr.Bytes())
		stderrLines := strings.Split(stderrStr, "\n")
		if len(stderrLines) < 3 {
			machine.StatusMessage = fmt.Sprintf("state code at %s didn't return at least 3 lines correctly at stderr (will keep retrying), stderr was: %s",
				cmdPath, stderrStr)
		} else {
			machine.LastState = machine.NextState
			machine.NextState = strings.TrimSpace(stderrLines[0])

			numSeconds, intConvertError := strconv.Atoi(strings.TrimSpace(stderrLines[1]))
			if intConvertError != nil || numSeconds <= 0 {
				machine.NextStateRun = time.Time{}
			} else {
				machine.NextStateRun = time.Now().Add(time.Duration(numSeconds) * time.Second)
			}

			machine.Input = string(stdout.Bytes())
			machine.StatusMessage = strings.TrimSpace(stderrLines[2])
		}
	}

	machine.RunningStateCode = false

	s.UpdatePersistedMachine(machine)

	if machine.NextState == "stop" {
		s.CancelMachineRun(fmt.Sprintf("%d", machine.Id))
	}
}

func (s *Scheduler) HandleTick() {
	s.SchedulerLock.Lock()

	currentTime := time.Now()

	for idx, machine := range s.RunningMachines {
		if !machine.RunningStateCode && machine.NextState != "stop" && machine.NextStateRun.Before(currentTime) {
			var machinePtr *RunningMachine = &(s.RunningMachines[idx])
			machinePtr.RunningStateCode = true
			s.UpdatePersistedMachine(machinePtr)
			go s.ExecuteState(machinePtr)
		}
	}

	s.SchedulerLock.Unlock()
}

func (s *Scheduler) SchedulerTick(ticker *time.Ticker, quitChannel chan struct{}) {
	for {
		select {
			case <- ticker.C:
				s.HandleTick()
			case <- quitChannel:
				ticker.Stop()
				return
		}
	}
}

func (s *Scheduler) Init(db *bolt.DB) chan struct{} {
	s.SchedulerLock = &sync.Mutex{}
	s.Database = db
	s.RunningMachines = make([]RunningMachine, 0, 0)

	// Initialize database
	dbInitErr := db.Update(func(tx *bolt.Tx) error {
		runningBucket, err := tx.CreateBucketIfNotExists([]byte("RunningMachines"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		var runsBucket *bolt.Bucket
		runsBucket, err = tx.CreateBucketIfNotExists([]byte("MachineRuns"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		c := runningBucket.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			machineJson := runsBucket.Get(k)

			var machine RunningMachine
			err := json.Unmarshal(machineJson, &machine)
			if err != nil {
				return fmt.Errorf("error deserializing currently running machine from persisted db: %s", err)
			}

			s.AddMachine(&machine)
		}

		return nil
	})

	if dbInitErr != nil {
		fmt.Printf("error initializing database: %s\n", dbInitErr)
		os.Exit(1)
	}


	// Setup the timer that powers the scheduler
	ticker := time.NewTicker(1 * time.Second)
	stopSchedulerChannel := make(chan struct{})
	go s.SchedulerTick(ticker, stopSchedulerChannel)
	return stopSchedulerChannel
}
