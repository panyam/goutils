package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gohttp "github.com/panyam/goutils/http"
)

type TimeHandler struct {
	Fanout *conc.FanOut[conc.Message[any]]
}

// ... along with a corresponding New method
func NewTimeHandler() *TimeHandler {
	return &TimeHandler{Fanout: conc.NewFanOut[conc.Message[any]](nil)}
}

// The Validate method gates the subscribe request to see if it should be upgraded
// and if so creates the right connection type to wrap the connection
// This examples allows all upgrades and is only needed to specify the kind of
// connection type to use - in this case TimeConn.
func (t *TimeHandler) Validate(w http.ResponseWriter, r *http.Request) (out *TimeConn, isValid bool) {
	return &TimeConn{handler: t}, true
}

// Our TimeConn allows us to override any connection instance specific behaviours
type TimeConn struct {
	gohttp.JSONConn
	handler *TimeHandler
}

func (t *TimeConn) OnStart(conn *websocket.Conn) error {
	t.JSONConn.OnStart(conn)
	writer := t.JSONConn.Writer

	log.Println("Got a new connection.....")
	// Register the writer channel into the Fanout
	t.handler.Fanout.Add(writer.SendChan(), nil, false)
	return nil
}

func (t *TimeConn) OnClose() {
	writer := t.JSONConn.Writer

	// Removal can be synchronous or asynchronous - we want to ensure it is done
	// synchronously so another publish (if one came in) wont be attempted on a closed channel
	<-t.handler.Fanout.Remove(writer.SendChan(), true)
	t.JSONConn.OnClose()
}

func (t *TimeConn) OnTimeout() bool {
	return false
}

func (t *TimeConn) HandleMessage(msg any) error {
	log.Println("Received Message To Handle: ", msg)
	// sending to all listeners
	t.handler.Fanout.Send(conc.Message[any]{Value: msg})
	return nil
}

func main() {
	r := mux.NewRouter()
	timeHandler := NewTimeHandler()
	r.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("msg")
		timeHandler.Fanout.Send(conc.Message[any]{Value: fmt.Sprintf("%s: %s", time.Now().String(), msg)})
		fmt.Fprintf(w, "Published Message Successfully")
	})

	// Send the time every 1 second
	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		for {
			<-t.C
			timeHandler.Fanout.Send(conc.Message[any]{Value: time.Now().String()})
		}
	}()

	r.HandleFunc("/subscribe", gohttp.WSServe(timeHandler, nil))
	srv := http.Server{Handler: r}
	log.Fatal(srv.ListenAndServe())
}
