package main

import "sync/atomic"

const (
	APPLICATION_STATE_INIT   = 0
	APPLICATION_STATE_STATED = 1
)

const (
	DRONE_HEALTH_UNKNOWN = 0
	DRONE_HEALTH_OK      = 1
	DRONE_HEALTH_NG      = 2
)

const (
	DRONE_STATE_UNKNOWN = 0
	DRONE_STATE_READY   = 1
	DRONE_STATE_LAND    = 2
	DRONE_STATE_TAKEOFF = 3
)

type ApplicationStates struct {
	applicationState atomic.Value
	currentStartKey  atomic.Value
	droneHealths     atomic.Value
	droneState       atomic.Value
}

type DroneHealths struct {
	DroneHealth  int
	BatteryLevel int
}

type DroneState int

func NewApplicationStates() *ApplicationStates {

	a := &ApplicationStates{}
	a.SetState(APPLICATION_STATE_INIT)
	a.SetStartKey("")
	a.SetDroneHealths(DroneHealths{
		DroneHealth: DRONE_HEALTH_UNKNOWN,
	})
	a.SetDroneState(DRONE_STATE_UNKNOWN)

	return a
}

func (a *ApplicationStates) GetState() int {
	return a.applicationState.Load().(int)
}

func (a *ApplicationStates) SetState(state int) {
	a.applicationState.Store(state)
}

func (a *ApplicationStates) GetStartKey() string {
	return a.currentStartKey.Load().(string)
}

func (a *ApplicationStates) SetStartKey(startKey string) {
	a.currentStartKey.Store(startKey)
}

func (a *ApplicationStates) GetDroneHealth() DroneHealths {
	return a.droneHealths.Load().(DroneHealths)
}

func (a *ApplicationStates) SetDroneHealths(healths DroneHealths) {
	a.droneHealths.Store(healths)
}

func (a *ApplicationStates) GetDroneState() DroneState {
	return a.droneState.Load().(DroneState)
}

func (a *ApplicationStates) SetDroneState(state DroneState) {
	a.droneState.Store(state)
}

func (a *ApplicationStates) IsStarted() bool {
	return a.GetState() == APPLICATION_STATE_STATED
}

func (a *ApplicationStates) Start() {
	a.SetState(APPLICATION_STATE_STATED)
}
