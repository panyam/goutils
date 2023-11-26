package ws

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gut "github.com/panyam/goutils/utils"
)

type WSHandler[I any, O any, S any] interface {
	/**
	 * Called to validate an http request to see if it is upgradeable to a ws conn
	 */
	Validate(w http.ResponseWriter, r *http.Request) (S, bool)

	/**
	 * Reads the next message from the ws conn.
	 */
	ReadMessage(state S, w *websocket.Conn) (I, error)

	/**
	 * Called to send the next ping message.
	 */
	SendPing(state S) error

	/**
	 * Called to handle the next message from the input stream on the ws conn.
	 */
	HandleMessage(state S, msg I) error

	/**
	 * On created.
	 */
	OnStart(state S, conn *websocket.Conn) error

	/**
	 * Called to handle or suppress an error
	 */
	OnError(state S, err error) error

	/**
	 * Called when the connection closes.
	 */
	OnClose(state S)
	OnTimeout(state S) bool
}

type JSONHandler[C any] struct {
	Writer *conc.Writer[conc.Message[interface{}]]
}

func (j *JSONHandler[S]) Validate(w http.ResponseWriter, r *http.Request) (out S, isValid bool) {
	// All connections upgradeable
	return
}

/**
 * Reads the next message from the ws conn.
 */
func (j *JSONHandler[S]) ReadMessage(state S, conn *websocket.Conn) (out interface{}, err error) {
	err = conn.ReadJSON(&out)
	return
}

/**
 * Called to send the next ping message.
 */
func (j *JSONHandler[S]) SendPing(state S) error {
	j.Writer.Send(conc.Message[interface{}]{Value: gut.StringMap{"type": "ping"}})
	return nil
}

/**
 * Called to handle the next message from the input stream on the ws conn.
 */
func (j *JSONHandler[S]) HandleMessage(state S, msg interface{}) error {
	log.Println("Received Message: ", msg)
	return nil
}

func (j *JSONHandler[S]) OnStart(state S, conn *websocket.Conn) error {
	j.Writer = conc.NewWriter(
		func(msg conc.Message[interface{}]) error {
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

func (j *JSONHandler[S]) OnError(state S, err error) error {
	return err
}

/**
 * Called when the connection closes.
 */
func (j *JSONHandler[S]) OnClose(state S) {
	// All the core hapens here
	if j.Writer != nil {
		j.Writer.Stop()
	}
}

func (j *JSONHandler[S]) OnTimeout(state S) bool {
	return true
}

type WSConnConfig struct {
	Upgrader   websocket.Upgrader
	PingPeriod time.Duration
	PongPeriod time.Duration
}

func DefaultWSConnConfig() *WSConnConfig {
	return &WSConnConfig{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		PingPeriod: time.Second * 30,
		PongPeriod: time.Second * 300,
	}
}

func WSServe[I any, O any, S any](h WSHandler[I, O, S], config *WSConnConfig) http.HandlerFunc {
	if config == nil {
		config = DefaultWSConnConfig()
	}
	return func(rw http.ResponseWriter, req *http.Request) {
		ctx, isValid := h.Validate(rw, req)
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

		WSHandleConn[I, O, S](conn, h, ctx, config)
	}
}

func WSHandleConn[I any, O any, S any](conn *websocket.Conn, h WSHandler[I, O, S], ctx S, config *WSConnConfig) {
	if config == nil {
		config = DefaultWSConnConfig()
	}
	reader := conc.NewReader[I](func() (I, error) {
		return h.ReadMessage(ctx, conn)
	})
	defer reader.Stop()

	lastReadAt := time.Now()
	pingTimer := time.NewTicker(config.PingPeriod)
	pongChecker := time.NewTicker(config.PongPeriod)
	defer pingTimer.Stop()
	defer pongChecker.Stop()

	defer h.OnClose(ctx)
	err := h.OnStart(ctx, conn)
	if err != nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(config.PongPeriod))
	h.SendPing(ctx)
	for {
		select {
		case <-pingTimer.C:
			h.SendPing(ctx)
			break
		case <-pongChecker.C:
			hb_delta := time.Now().Sub(lastReadAt).Seconds()
			if hb_delta > config.PongPeriod.Seconds() {
				// Lost connection with conn so can drop off?
				if h.OnTimeout(ctx) {
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
					if h.OnError(ctx, result.Error) != nil {
						log.Println("Unknown Error: ", result.Error)
						return
					}
				}
			} else {
				// we have an actual message being sent on this channel - typically
				// dont need to do anything as we are using these for outbound connections
				// only to write to a listening agent FE so can just log and drop any
				// thing sent by agent FE here - this can change later
				h.HandleMessage(ctx, result.Value)
			}
			break
		}
	}
}
