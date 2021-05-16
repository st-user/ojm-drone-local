package main

import (
	"log"
	"net"
	"strings"
	"time"
)

type Drone struct {
	commands map[string]string
}

func NewDrone() Drone {

	MOVE_SPEED := "30"
	ROTATION_SPEED := "30"
	commands := make(map[string]string)
	commands["forward"] = "forward " + MOVE_SPEED
	commands["back"] = "back " + MOVE_SPEED
	commands["right"] = "right " + MOVE_SPEED
	commands["left"] = "left " + MOVE_SPEED

	commands["up"] = "up " + ROTATION_SPEED
	commands["down"] = "down " + ROTATION_SPEED
	commands["cw"] = "cw " + ROTATION_SPEED
	commands["ccw"] = "ccw " + ROTATION_SPEED

	return Drone{
		commands: commands,
	}
}

func sendCommand(command string, conn net.Conn) {
	_, err := conn.Write([]byte(command))
	if err != nil {
		log.Println(err)
	}
}

func (drone *Drone) sendCommandFromMessage(message string, conn *net.Conn) {
	command, ok := drone.commands[message]
	if !ok {
		command = message
	}
	sendCommand(command, *conn)
}

func (drone *Drone) Start(routineCoordinator *RoutineCoordinator) {

	pingTicker := time.NewTicker(10 * time.Second)
	stateInfoChannel := make(chan map[string]string)
	go func() {
		commandConn, err := net.Dial("udp", "192.168.10.1:8889")
		if err != nil {
			log.Println(err)
			routineCoordinator.StopApp()
			return
		}
		defer commandConn.Close()
		defer pingTicker.Stop()

		sendCommand("command", commandConn)
		sendCommand("streamon", commandConn)

		for {
			select {
			case command := <-routineCoordinator.DroneCommandChannel:
				drone.sendCommandFromMessage(command, &commandConn)
			case <-pingTicker.C:
				sendCommand("command", commandConn)
				sendCommand("streamon", commandConn)
			case stateInfo := <-stateInfoChannel:
				log.Printf("Battery level %v%%", stateInfo["bat"])
			case <-routineCoordinator.StopSignalChannel:
				log.Println("Stop drone event loop.")
				return
			}

		}
	}()

	go func() {
		defer close(stateInfoChannel)
		drone.parseSteteResponse(&stateInfoChannel, routineCoordinator)
	}()

	go func() {
		videoConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 11111})
		if err != nil {
			log.Println(err)
			routineCoordinator.StopApp()
			return
		}

		routineCoordinator.AddWaitGroupUntilReleasingSocket()
		defer func() {
			videoConn.Close()
			log.Println("Connection for video streaming has closed.")
			routineCoordinator.DoneWaitGroupUntilReleasingSocket()
		}()

		inboundUDPPacket := make([]byte, 1600) // UDP MTU

		var buf []byte
		var frame []byte
		isNalUnitStart := func(b []byte) bool {
			return len(b) > 3 && b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 1
		}

		sendPreviousBytes := func(b []byte) bool {
			return len(b) > 4 && (b[4]&0b11111 == 7 || b[4]&0b11111 == 1)
		}

		for {

			select {
			case <-routineCoordinator.StopSignalChannel:
				log.Println("Stop capturing video stream.")
				return
			default:
				n, _, err := videoConn.ReadFrom(inboundUDPPacket)
				if err != nil {
					log.Println(err)
					continue
				}
				data := inboundUDPPacket[:n]

				if !isNalUnitStart(data) || !sendPreviousBytes(data) {
					buf = append(buf, data...)
					continue
				} else {
					frame = buf
					var zero []byte
					buf = append(zero, data...)
				}

				if len(frame) == 0 {
					continue
				}

				routineCoordinator.SendDroneFrameChannel(&frame)
			}

		}
	}()
}

func (drone *Drone) parseSteteResponse(stateInfoChannel *chan map[string]string, routineCoordinator *RoutineCoordinator) {
	stateConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8890})
	if err != nil {
		log.Println(err)
		routineCoordinator.StopApp()
		return
	}

	routineCoordinator.AddWaitGroupUntilReleasingSocket()
	defer func() {
		stateConn.Close()
		log.Println("Connection for checking drone state has closed.")
		routineCoordinator.DoneWaitGroupUntilReleasingSocket()
	}()
	inboundUDPPacket := make([]byte, 1600) // UDP MTU
	last := time.Now()
	for {

		select {
		case <-routineCoordinator.StopSignalChannel:
			log.Println("Stop parsing drone state.")
			return
		default:

			n, _, err := stateConn.ReadFrom(inboundUDPPacket)
			if err != nil {
				log.Println(err)
				continue
			}
			duration := time.Since(last)
			if duration.Seconds() < 10 {
				continue
			}
			last = time.Now()
			data := inboundUDPPacket[:n]
			result := make(map[string]string)
			elems := strings.Split(string(data), ";")
			for _, elem := range elems {
				kv := strings.Split(elem, ":")
				if len(kv) > 1 {
					result[kv[0]] = kv[1]
				}
			}
			*stateInfoChannel <- result
		}
	}
}
