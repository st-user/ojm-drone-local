package main

import (
	"sync"

	"github.com/pion/rtcp"
)

type RoutineCoordinator struct {
	DroneCommandChannel           chan DroneCommand
	DroneFrameChannel             chan []byte
	DataChannelMessageChannel     chan string
	RTCPPacketChannel             chan rtcp.Packet
	StopSignalChannel             chan struct{}
	IsStopped                     bool
	waitGroupUntilReleasingSocket sync.WaitGroup
	mutex                         sync.Mutex
}

type DroneCommand struct {
	CommandType string
	Command     interface{}
}

type MotionVector struct {
	X float32
	Y float32
	Z float32
	R float32
}

func (r *RoutineCoordinator) InitRoutineCoordinator(force bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.IsStopped || force {
		r.DroneCommandChannel = make(chan DroneCommand)
		r.DroneFrameChannel = make(chan []byte)
		r.DataChannelMessageChannel = make(chan string)
		r.RTCPPacketChannel = make(chan rtcp.Packet)
		r.StopSignalChannel = make(chan struct{})
	}
	r.IsStopped = false
}

func (r *RoutineCoordinator) StopApp() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.IsStopped {
		return
	}

	r.IsStopped = true
	close(r.DroneCommandChannel)
	close(r.DroneFrameChannel)
	close(r.DataChannelMessageChannel)
	close(r.RTCPPacketChannel)
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

func (r *RoutineCoordinator) SendDroneCommandChannel(data DroneCommand) {
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

func (r *RoutineCoordinator) SendRTCPPacketChannel(data rtcp.Packet) {
	if !r.IsStopped {
		r.RTCPPacketChannel <- data
	}
}

func (r *RoutineCoordinator) SendStopSignalChannel(data struct{}) {
	if !r.IsStopped {
		r.StopSignalChannel <- data
	}
}

func (mVec *MotionVector) isZeroVector() bool {
	return mVec.X == 0 && mVec.Y == 0 && mVec.Z == 0 && mVec.R == 0
}
