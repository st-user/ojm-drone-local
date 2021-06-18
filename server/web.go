package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/st-user/ojm-drone-local/applog"
	"github.com/st-user/ojm-drone-local/appos"
)

var mimeTypes = map[string]string{
	".html": "text/html",
	".js":   "text/javascript",
	".css":  "text/css",
	".json": "application/json",
	".png":  "image/png",
	".jpg":  "image/jpg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".wav":  "audio/wav",
	".mp4":  "video/mp4",
	".woff": "application/font-woff",
	".ttf":  "application/font-ttf",
	".eot":  "application/vnd.ms-fontobject",
	".otf":  "application/font-otf",
	".wasm": "application/wasm",
}

type Statics struct {
	dir string
}

func NewStatics() Statics {
	dir := filepath.Join(appos.BaseDir(), "static")
	_dir := os.Getenv("GO_STATIC_FILE_DIR")

	if len(_dir) > 0 {
		dir = _dir
	}

	return Statics{
		dir: dir,
	}
}

func (s *Statics) HandleStatic(w http.ResponseWriter, r *http.Request) {

	filename := r.URL.Path[len("/"):]
	ext := filepath.Ext(filename)
	mimeType, ok := mimeTypes[ext]

	if !ok {
		mimeType = "application/octet-stream"
	}

	if filename == "" {
		filename = "index.html"
		mimeType = "text/html"
	}

	path := filepath.Join(s.dir, filename)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	_body, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type", mimeType)
	w.Write(_body)
}

func HandleFuncJSON(
	path string,
	handler func(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error)) {

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {

		applog.Info("Request to %v", path)

		result, err := handler(w, r)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			applog.Warn("%v", err)
			WriteInternalServerError(w, &err)
			return
		}

		json.NewEncoder(w).Encode(*result)
	})
}

func WriteInternalServerError(w http.ResponseWriter, err *error) {
	w.WriteHeader(500)
	applog.Info("%v", err)
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
		WriteInternalServerError(w, &err)
		return
	}
	applog.Info("Connected.")

	go func() {
		defer conn.Close()

		for {

			select {
			case text := <-routineCoordinator.DroneStateChannel:
				stateJson := messageHandler(text)
				if err := conn.WriteJSON(stateJson); err != nil {
					applog.Info("%v", err)
					continue
				}
			case <-routineCoordinator.StopSignalChannel:
				applog.Info("Stop OutboundRelayMessageServer")
				return
			}

		}

	}()
}
