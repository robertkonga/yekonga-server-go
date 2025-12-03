package Yekonga

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/robertkonga/yekonga-server/plugins/socketio"
	"github.com/robertkonga/yekonga-server/plugins/socketio/engineio"
	"github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport"
	"github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport/polling"
	"github.com/robertkonga/yekonga-server/plugins/socketio/engineio/transport/websocket"
)

// Easier to get running with CORS. Thanks for help @Vindexus and @erkie
var allowOriginFunc = func(r *http.Request) bool {
	return true
}

func NewSocketServer(y *YekongaData) *socketio.Server {

	socketServer := socketio.NewServer(&engineio.Options{
		PingTimeout:  80 * time.Second,
		PingInterval: 40 * time.Second,
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
		// Transports: []transport.Transport{
		// 	websocket.Default,
		// 	polling.Default,
		// },
		RequestChecker: func(req *http.Request) (http.Header, error) {
			// req.Header["access-control-allow-headers"] = []string{"content-type, authorization, timezone, upgrade-insecure-requests"}

			return req.Header, nil
		},
	})

	// Handle new connections
	socketServer.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("New client connected:", s.ID())

		s.Emit("message", "Welcome to the Go Socket.IO server!")

		return nil
	})

	socketServer.OnError("/", func(s socketio.Conn, err error) {
		fmt.Println("Error", err)
	})

	// Handle custom events
	socketServer.OnEvent("/", "chat_message", func(s socketio.Conn, msg string) {
		fmt.Println("Received message:", msg)
		s.Emit("chat_reply", "You said: "+msg)
	})

	// Handle disconnections
	socketServer.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("Client disconnected:", s.ID(), reason)
	})

	go func() {
		if err := socketServer.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		} else {
			log.Fatalf("socketio listen success")
		}
	}()

	return socketServer
}
