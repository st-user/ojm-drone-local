package main

import "sync/atomic"

const (
	DRONE_HEALTH_UNKNOWN = 0
	DRONE_HEALTH_OK      = 1
	DRONE_HEALTH_NG      = 2
)

type ApplicationStates struct {
	DroneStates *DroneStates
}

type DroneStates struct {
	droneHealth  atomic.Value
	batteryLevel atomic.Value
}

func NewApplicationStates() *ApplicationStates {

	droneStates := DroneStates{}
	droneStates.SetDroneHealth(DRONE_HEALTH_UNKNOWN)
	droneStates.SetBatteryLevel(0)

	return &ApplicationStates{
		DroneStates: &droneStates,
	}
}

func (v *DroneStates) DroneHealth() int {
	return v.droneHealth.Load().(int)
}

func (v *DroneStates) SetDroneHealth(flg int) {
	v.droneHealth.Store(flg)
}

func (v *DroneStates) BatteryLevel() int8 {
	return v.batteryLevel.Load().(int8)
}

func (v *DroneStates) SetBatteryLevel(batteryLevel int8) {
	v.batteryLevel.Store(batteryLevel)
}
