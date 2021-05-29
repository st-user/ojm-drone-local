module github.com/st-user/ojm-drone-local

go 1.16

require (
	github.com/gorilla/websocket v1.4.2
	github.com/pion/rtcp v1.2.6
	github.com/pion/webrtc/v3 v3.0.29
	gobot.io/x/gobot v0.0.0-00010101000000-000000000000
)

replace gobot.io/x/gobot => ../../gobot
