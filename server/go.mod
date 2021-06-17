module github.com/st-user/ojm-drone-local

go 1.16

require (
	github.com/danieljoos/wincred v1.1.0
	github.com/gorilla/websocket v1.4.2
	github.com/keybase/go-keychain v0.0.0-20201121013009-976c83ec27a6
	github.com/pion/rtcp v1.2.6
	github.com/pion/webrtc/v3 v3.0.29
	gobot.io/x/gobot v0.0.0-00010101000000-000000000000
)

replace gobot.io/x/gobot => ../../gobot
