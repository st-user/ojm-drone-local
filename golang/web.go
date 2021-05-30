package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
)

type Statics struct {
	dir      string
	contents map[string][]byte
}

func NewStatics() Statics {
	dir := "static"
	_dir := os.Getenv("GO_STATIC_FILE_DIR")

	if len(_dir) > 0 {
		dir = _dir
	}

	return Statics{
		dir:      dir,
		contents: make(map[string][]byte),
	}
}

func (s *Statics) ToHandleFunc(filename string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := s.contents[filename]
		if !ok {
			path := filepath.Join(s.dir, filename)

			_body, err := ioutil.ReadFile(path)

			if err != nil {
				log.Fatal(err)
			}
			_copy := make([]byte, len(_body))
			copy(_copy, _body)
			s.contents[filename] = _copy
			body = _body
		}
		w.Write(body)
	}
}

func HandleFuncJSON(
	path string,
	handler func(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error)) {

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {

		Log.Info("Request to %v", path)

		result, err := handler(w, r)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			writeInternalServerError(w, &err)
			return
		}

		json.NewEncoder(w).Encode(*result)
	})
}

func writeInternalServerError(w http.ResponseWriter, err *error) {
	w.WriteHeader(500)
	Log.Info("%v", err)
}

type OutboundRelayMessageServer struct {
	upgrader websocket.Upgrader
}

func NewOutboundRelayMessageServer() OutboundRelayMessageServer {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	return OutboundRelayMessageServer{
		upgrader: upgrader,
	}
}

func (ws *OutboundRelayMessageServer) HandleMessage(
	w http.ResponseWriter,
	r *http.Request,
	routineCoordinator *RoutineCoordinator,
	messageHandler func(text string) map[string]interface{}) {

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		writeInternalServerError(w, &err)
		return
	}
	Log.Info("Connected.")

	go func() {
		defer conn.Close()

		for {

			select {
			case text := <-routineCoordinator.DroneStateChannel:
				stateJson := messageHandler(text)
				if err := conn.WriteJSON(stateJson); err != nil {
					Log.Info("%v", err)
					continue
				}
			case <-routineCoordinator.StopSignalChannel:
				Log.Info("Stop OutboundRelayMessageServer")
				return
			}

		}

	}()
}
