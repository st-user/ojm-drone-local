package main

import (
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/st-user/ojm-drone-local/applog"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
)

type Drone struct {
	driver       *tello.Driver
	safetySignal SafetySignal
}

func NewDrone() Drone {
	return Drone{
		safetySignal: NewSafetySignal(),
	}
}

func (drone *Drone) Start(routineCoordinator *RoutineCoordinator, applicationStates *ApplicationStates) {

	var driver *tello.Driver
	var robot *gobot.Robot

	lastTimestampVideoReceived := time.Now().Add(-1 * time.Hour)
	lastTimestampFightDataReceived := time.Now().Add(-1 * time.Hour)
	latestBatteryLevel := 0

	startRobot := func() {

		driver = tello.NewDriver("8888")
		drone.driver = driver

		driver.On(tello.ConnectedEvent, func(data interface{}) {
			applog.Info("Starts receiving video frames from your drone.")
			driver.StartVideo()
			driver.SetVideoEncoderRate(tello.VideoBitRate1M)
			gobot.Every(10*time.Second, func() {
				driver.StartVideo()
			})
		})

		lastLoggedTime := time.Now()
		driver.On(tello.FlightDataEvent, func(data interface{}) {
			lastTimestampFightDataReceived = time.Now()

			if 3 < time.Since(lastLoggedTime).Seconds() {
				fd := data.(*tello.FlightData)

				latestBatteryLevel = int(fd.BatteryPercentage)
				applog.Info("Battery level %v%%", fd.BatteryPercentage)

				lastLoggedTime = time.Now()
			}
		})

		// Thanks to [oliverpool/tello-webrtc-fpv](https://github.com/oliverpool/tello-webrtc-fpv)
		// I was able to figure out the timing at which h264 packets should be send to a browser.
		var buf []byte
		isNalUnitStart := func(b []byte) bool {
			return len(b) > 3 && b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 1
		}

		sendPreviousBytes := func(b []byte) bool {
			return len(b) > 4 && (b[4]&0b11111 == 7 || b[4]&0b11111 == 1)
		}

		loggedRecoverCount := 0
		handleData := func(_data interface{}) {
			lastTimestampVideoReceived = time.Now()

			defer func() {
				if r := recover(); r != nil {
					if loggedRecoverCount%100 == 0 {
						applog.Info("Ignores panic. %v", r)
						loggedRecoverCount = 0
					}
					loggedRecoverCount++
				}
			}()

			data := _data.([]byte)

			if !isNalUnitStart(data) || !sendPreviousBytes(data) {
				buf = append(buf, data...)
				return
			} else {
				routineCoordinator.SendDroneFrameChannel(&buf)
				var zero []byte
				buf = append(zero, data...)
			}

		}
		driver.On(tello.VideoFrameEvent, handleData)
		robot = gobot.NewRobot(
			[]gobot.Connection{},
			[]gobot.Device{driver},
		)
		robot.AutoRun = false
		robot.Start()
	}

	go func() {
		routineCoordinator.AddWaitGroupUntilReleasingSocket()
		defer routineCoordinator.DoneWaitGroupUntilReleasingSocket()

		for {

			select {
			case command := <-routineCoordinator.DroneCommandChannel:
				switch command.CommandType {
				case "takeoff":
					drone.driver.TakeOff()
				case "land":
					drone.driver.Land()
				case "vector":
					mVec := command.Command.(MotionVector)
					drone.safetySignal.ConsumeSignal(mVec, drone)
					drone.driver.SetVector(mVec.Y, mVec.X, mVec.Z, mVec.R)
				}

			case pkt := <-routineCoordinator.RTCPPacketChannel:

				switch _pkt := pkt.(type) {
				case *rtcp.PictureLossIndication:
					applog.Debug("Receives RTCP PictureLossIndication. %v", _pkt)
					drone.driver.StartVideo()

				case *rtcp.ReceiverEstimatedMaximumBitrate:
					applog.Debug("Receives RTCP ReceiverEstimatedMaximumBitrate. %v", _pkt)
					bitrate := float64(_pkt.Bitrate)

					// Using the bitrate(MB) value corresponding to the one that 'rtcp.Receiver Estimated Maximum Bitrate.String()' shows.
					// Reference: github.com/pion/rtcp receiver_estimated_maximum_bitrate.go
					bitrateMB := bitrate / 1000.0 / 1000.0 // :MB
					var changeTo float64

					switch {
					case bitrateMB >= 4.0:
						drone.driver.SetVideoEncoderRate(tello.VideoBitRate4M)
						changeTo = 4.0
					case bitrateMB >= 3.0:
						drone.driver.SetVideoEncoderRate(tello.VideoBitRate3M)
						changeTo = 3.0
					case bitrateMB >= 2.0:
						drone.driver.SetVideoEncoderRate(tello.VideoBitRate2M)
						changeTo = 2.0
					case bitrateMB >= 1.5:
						drone.driver.SetVideoEncoderRate(tello.VideoBitRate15M)
						changeTo = 1.5
					default:
						drone.driver.SetVideoEncoderRate(tello.VideoBitRate1M)
						changeTo = 1
					}
					applog.Debug("ReceiverEstimation = %.2f Mb/s. The bit rate changes to %v Mb/s", bitrateMB, changeTo)
				}

			case <-routineCoordinator.StopSignalChannel:
				applog.Info("Stop drone event loop.")
				robot.Stop()
				return
			}

		}
	}()

	checkerFunc := func() {
		routineCoordinator.AddWaitGroupUntilReleasingSocket()
		defer routineCoordinator.DoneWaitGroupUntilReleasingSocket()

		for {
			select {
			case <-routineCoordinator.StopSignalChannel:
				applog.Info("Stop drone health check loop.")

				applicationStates.SetDroneHealths(DroneHealths{
					DroneHealth:  DRONE_HEALTH_UNKNOWN,
					BatteryLevel: latestBatteryLevel,
				})

				robot.Stop()
				return
			default:

				ok := true
				if time.Since(lastTimestampVideoReceived).Seconds() > 5 {
					applog.Warn("Drone fails to receive video stream.")
					ok = false
				}

				if time.Since(lastTimestampFightDataReceived).Seconds() > 5 {
					applog.Warn("Drone fails to receive flight data.")
					ok = false
				}

				if ok {

					applicationStates.SetDroneHealths(DroneHealths{
						DroneHealth:  DRONE_HEALTH_OK,
						BatteryLevel: latestBatteryLevel,
					})

					applog.Debug("Drone is successfully receiving data.")
				} else {

					applicationStates.SetDroneHealths(DroneHealths{
						DroneHealth:  DRONE_HEALTH_NG,
						BatteryLevel: latestBatteryLevel,
					})

					robot.Stop()
					applog.Info("Restarts robot.")
					startRobot()
					time.Sleep(1 * time.Second)
				}

				time.Sleep(1 * time.Second)
			}
		}
	}

	startRobot()
	go checkerFunc()

	applog.Info("Drone starts.")
}

// In case of losing a stop signal (i.e '{ x: 0, y: 0 }' or '{ r: 0, z: 0 }') for some reason,
// if no signal is received during 500ms, a stop signal is set automatically
type SafetySignal struct {
	isStarted             bool
	endChannel            chan struct{}
	lastAccessedTimestamp time.Time
	mutex                 sync.Mutex
}

func NewSafetySignal() SafetySignal {
	return SafetySignal{}
}

func (s *SafetySignal) ConsumeSignal(mVec MotionVector, drone *Drone) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.startChecking(drone)

	if mVec.isZeroVector() {
		s.endChecking()
		return
	}
	s.lastAccessedTimestamp = time.Now()
}

func (s *SafetySignal) startChecking(drone *Drone) {
	if s.isStarted {
		return
	}
	s.endChannel = make(chan struct{})
	s.lastAccessedTimestamp = time.Now()
	s.isStarted = true
	go func() {

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.endChannel:
				return
			case <-ticker.C:
				if 500 < time.Since(s.lastAccessedTimestamp).Milliseconds() {
					s.mutex.Lock()
					defer s.mutex.Unlock()

					applog.Info("Set a zero translation vector because of losing a stop signal.")
					drone.driver.SetVector(0, 0, 0, 0)
					s.endChecking()
					return
				}
			}
		}
	}()
}

func (s *SafetySignal) endChecking() {
	s.isStarted = false
	s.lastAccessedTimestamp = time.Now()
	close(s.endChannel)
}
