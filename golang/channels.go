package main

import (
	"sync"
)

type RoutineCoordinator struct {
	DroneCommandChannel           chan string
	DroneFrameChannel             chan []byte
	DataChannelMessageChannel     chan string
	DroneStateChannel             chan string
	StopSignalChannel             chan struct{}
	IsStopped                     bool
	waitGroupUntilReleasingSocket sync.WaitGroup
}

func (r *RoutineCoordinator) InitRoutineCoordinator(force bool) {

	if r.IsStopped || force {
		r.DroneCommandChannel = make(chan string)
		r.DroneFrameChannel = make(chan []byte)
		r.DataChannelMessageChannel = make(chan string)
		r.DroneStateChannel = make(chan string)
		r.DroneCommandChannel = make(chan string)
		r.StopSignalChannel = make(chan struct{})
	}
	r.IsStopped = false
}

func (r *RoutineCoordinator) StopApp() {
	r.IsStopped = true
	close(r.DroneCommandChannel)
	close(r.DroneFrameChannel)
	close(r.DataChannelMessageChannel)
	close(r.DroneStateChannel)
	close(r.StopSignalChannel)
}

func (r *RoutineCoordinator) WaitUntilReleasingSocket() {
	r.waitGroupUntilReleasingSocket.Wait()
}

func (r *RoutineCoordinator) AddWaitGroupUntilReleasingSocket() {
	r.waitGroupUntilReleasingSocket.Add(1)
}

func (r *RoutineCoordinator) DoneWaitGroupUntilReleasingSocket() {
	r.waitGroupUntilReleasingSocket.Done()
}

func (r *RoutineCoordinator) ChangeDroneState(command string) {
	r.DroneCommandChannel <- command
	r.DataChannelMessageChannel <- command
}

func (r *RoutineCoordinator) SendDroneCommandChannel(data string) {
	if !r.IsStopped {
		r.DroneCommandChannel <- data
	}
}

func (r *RoutineCoordinator) SendDroneFrameChannel(data *[]byte) {
	if !r.IsStopped {
		r.DroneFrameChannel <- *data
	}
}

func (r *RoutineCoordinator) SendDataChannelMessageChannel(data string) {
	if !r.IsStopped {
		r.DataChannelMessageChannel <- data
	}
}

func (r *RoutineCoordinator) SendDroneStateChannel(data string) {
	if !r.IsStopped {
		r.DroneStateChannel <- data
	}
}

func (r *RoutineCoordinator) SendStopSignalChannel(data struct{}) {
	if !r.IsStopped {
		r.StopSignalChannel <- data
	}
}
