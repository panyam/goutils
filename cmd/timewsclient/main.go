package main

import (
	"log"

	"github.com/gorilla/websocket"
	gohttp "github.com/panyam/goutils/http"
)

type TimeHandler struct {
}

// ... along with a corresponding New method
func NewTimeHandler() *TimeHandler {
	return &TimeHandler{}
}

// Our TimeClientConn allows us to override any connection instance specific behaviours
type TimeClientConn struct {
	gohttp.JSONConn
}

func (t *TimeClientConn) HandleMessage(msg any) error {
	// Just print it
	log.Println("Received Message To Handle: ", msg)
	return nil
}

func main() {

	// create a (gorilla) websocker dialer
	dialer := *websocket.DefaultDialer

	// and dial - ignoring errors for now
	conn, _, err := dialer.Dial("ws://localhost/subscribe", nil)
	if err != nil {
		panic(err)
	}

	var timeconn TimeClientConn
	gohttp.WSHandleConn(conn, &timeconn, nil)
}
