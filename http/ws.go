package http

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gut "github.com/panyam/goutils/utils"
)

// Represents a bidirectional websocket connection
type WSConn[I any] interface {
	BiDirStreamConn[I]

	// Reads the next message from the ws conn.
	ReadMessage(w *websocket.Conn) (I, error)

	// Callback to be called when the WS connection is started
	OnStart(conn *websocket.Conn) error
}

// Handlers validate a http request and decide whether they can/should be upgraded
// to create and begin a websocket connection (WSConn)
type WSHandler[I any, S WSConn[I]] interface {
	// Called to validate an http request to see if it is upgradeable to a ws conn
	Validate(w http.ResponseWriter, r *http.Request) (S, bool)
}

// Extends BiDirStreamConfig to include Websocket specific configrations
type WSConnConfig struct {
	*BiDirStreamConfig
	Upgrader websocket.Upgrader
}

// This method creates a WSConnConfig with a default websocket Upgrader
func DefaultWSConnConfig() *WSConnConfig {
	return &WSConnConfig{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		BiDirStreamConfig: DefaultBiDirStreamConfig(),
	}
}

// Returns a http.HandlerFunc that takes care of upgrading the request to a Websocket connection
// and handling its lifecycle by delegating important activities to the WSHandler type
// This method is often used to create a handler for particular routes on http routers.
//
// The handler parameter is responsible for validating (eg authenticating/authorizing) the request
// to ensure an upgrade is allowed as well as handling messages received on the upgraded connection.
func WSServe[I any, S WSConn[I]](handler WSHandler[I, S], config *WSConnConfig) http.HandlerFunc {
	if config == nil {
		config = DefaultWSConnConfig()
	}
	return func(rw http.ResponseWriter, req *http.Request) {
		ctx, isValid := handler.Validate(rw, req)
		if !isValid {
			return
		}

		// Standard upgrade to WS .....
		conn, err := config.Upgrader.Upgrade(rw, req, nil)
		if err != nil {
			http.Error(rw, "WS Upgrade failed", 400)
			log.Println("WS upgrade failed: ", err)
			return
		}
		defer conn.Close()

		log.Println("Start Handling Conn with: ", ctx)
		WSHandleConn(conn, ctx, config)
	}
}

// Once a websocket connection is established (either by the server or by the client),
// this method handles the lifecycle of the connection by taking care of (healthceck) pings,
// handling closures, handling received messages.
func WSHandleConn[I any, S WSConn[I]](conn *websocket.Conn, ctx S, config *WSConnConfig) {
	if config == nil {
		config = DefaultWSConnConfig()
	}
	reader := conc.NewReader(func() (I, error) {
		res, err := ctx.ReadMessage(conn)
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
			return res, net.ErrClosed
		}
		return res, err
	})
	defer reader.Stop()

	lastReadAt := time.Now()
	pingTimer := time.NewTicker(config.PingPeriod)
	pongChecker := time.NewTicker(config.PongPeriod)
	defer pingTimer.Stop()
	defer pongChecker.Stop()

	defer ctx.OnClose()
	err := ctx.OnStart(conn)
	if err != nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(config.PongPeriod))
	for {
		select {
		case <-pingTimer.C:
			ctx.SendPing()
			break
		case <-pongChecker.C:
			hb_delta := time.Now().Sub(lastReadAt).Seconds()
			if hb_delta > config.PongPeriod.Seconds() {
				// Lost connection with conn so can drop off?
				if ctx.OnTimeout() {
					log.Printf("Last heart beat more than %d seconds ago.  Killing connection", int(hb_delta))
					return
				}
			}
			break
		case result := <-reader.RecvChan():
			conn.SetReadDeadline(time.Now().Add(config.PongPeriod))
			lastReadAt = time.Now()
			if result.Error != nil {
				if result.Error != io.EOF {
					if ce, ok := result.Error.(*websocket.CloseError); ok {
						log.Println("WebSocket Closed: ", ce)
						switch ce.Code {
						case websocket.CloseAbnormalClosure:
						case websocket.CloseNormalClosure:
						case websocket.CloseGoingAway:
							return
						}
					}
					if ctx.OnError(result.Error) != nil {
						log.Println("Closing due to Error: ", result.Error)
						return
					}
				}
			} else {
				// we have an actual message being sent on this channel - typically
				// dont need to do anything as we are using these for outbound connections
				// only to write to a listening agent FE so can just log and drop any
				// thing sent by agent FE here - this can change later
				ctx.HandleMessage(result.Value)
			}
			break
		}
	}
}

// An implementation of the WSConn interface that exchanges JSON message paylods
type JSONConn struct {
	// The output writer as a channel to send outgoing messages on
	Writer *conc.Writer[conc.Message[any]]

	// Name of this connection (for clarity)
	NameStr string

	// A connection ID to identify this connection
	ConnIdStr string

	// Keeps track of the current Ping count to send with the ping
	PingId int64
}

// Gets the name of this connection
func (j *JSONConn) Name() string {
	if j.NameStr == "" {
		j.NameStr = "JSONConn"
	}
	return j.NameStr
}

// Basic debug information.
func (j *JSONConn) DebugInfo() any {
	return map[string]any{
		"writer": j.Writer.DebugInfo(),
		"Name":   j.NameStr,
		"ConnId": j.ConnIdStr,
		"PingId": j.PingId,
	}
}

// Returns the (possibly auto-generated) Connection Id
func (j *JSONConn) ConnId() string {
	if j.ConnIdStr == "" {
		j.ConnIdStr = gut.RandString(10, "")
	}
	return j.ConnIdStr
}

// Reads the next message from the websocket connection as a JSON payload
func (j *JSONConn) ReadMessage(conn *websocket.Conn) (out any, err error) {
	err = conn.ReadJSON(&out)
	return
}

// This (callback) method is called when the websocket connection is upgraded but before
// the websocket event loop begins (in the WSHandleConn method)
func (j *JSONConn) OnStart(conn *websocket.Conn) error {
	log.Printf("Starting %s connection: %s", j.Name(), j.ConnId()) // E: e.connId undefined (type *HubConn has n…
	j.Writer = conc.NewWriter(
		func(msg conc.Message[any]) error {
			if msg.Error == io.EOF {
				log.Println("Streamer closed...", msg.Error)
				// do nothing
				// SendJsonResponse(rw, nil, msg.Error)
				return nil
			} else if msg.Error != nil {
				return WSConnWriteError(conn, msg.Error)
			} else {
				return WSConnWriteMessage(conn, msg.Value)
			}
		})
	return nil
}

// Called to send the next ping message.
func (j *JSONConn) SendPing() error {
	j.PingId += 1
	j.Writer.Send(conc.Message[any]{Value: gut.StrMap{
		"type":   "ping",
		"pingId": j.PingId,
		"name":   j.Name(),
		"connId": j.ConnId(),
	}})
	return nil
}

// Called to handle the next message from the input stream on the ws conn.
func (j *JSONConn) HandleMessage(msg any) error {
	log.Println("Received Message: ", msg)
	return nil
}

func (j *JSONConn) OnError(err error) error {
	return err
}

// Called when the connection closes.
func (j *JSONConn) OnClose() {
	// All the core hapens here
	if j.Writer != nil {
		j.Writer.Stop()
	}
	log.Printf("Closed %s connection: %s", j.Name(), j.ConnId()) // E: e.connId undefined (type *HubConn has n…
}

// Handle read timeouts.  By default returns true to disconnect and close on a timeout.
func (j *JSONConn) OnTimeout() bool {
	return true
}

type JSONHandler struct {
}

func (j *JSONHandler) Validate(w http.ResponseWriter, r *http.Request) (out WSConn[any], isValid bool) {
	// All connections upgradeable
	return &JSONConn{}, true
}

/*
type DefaultWSHandler[I any] struct {
}

// Called to validate an http request to see if it is upgradeable to a ws conn
func (d *DefaultWSHandler[I]) Validate(w http.ResponseWriter, r *http.Request) (WSConn[I], bool) {
	return nil, true
}
*/
