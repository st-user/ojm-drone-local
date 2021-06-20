package main

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

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
	DRONE_STATE_INIT    = 0
	DRONE_STATE_READY   = 1
	DRONE_STATE_LAND    = 2
	DRONE_STATE_TAKEOFF = 3
)

const (
	SESSION_KEY_HTTP_HEADER_KEY = "x-ojm-drone-local-session-key"
)

type ApplicationStates struct {
	applicationState atomic.Value
	currentStartKey  atomic.Value
	droneHealths     atomic.Value
	droneState       atomic.Value
	sessionKey       atomic.Value
	StartStopMux     sync.Mutex
	AccessKey        string
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
	a.SetDroneState(DRONE_STATE_INIT)
	a.ChangeSessionKey()

	key, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	a.AccessKey = key.String()

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

func (a *ApplicationStates) GetSessionKey() string {
	return a.sessionKey.Load().(string)
}

func (a *ApplicationStates) SetSessionKey(sessionKey string) {
	a.sessionKey.Store(sessionKey)
}

func (a *ApplicationStates) ChangeSessionKey() {
	key, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	a.SetSessionKey(key.String())
}

func (a *ApplicationStates) IsStarted() bool {
	return a.GetState() == APPLICATION_STATE_STATED
}

func (a *ApplicationStates) Start() {
	a.SetState(APPLICATION_STATE_STATED)
}
