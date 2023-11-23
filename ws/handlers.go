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

type WSHandler[I any, O any, C any] interface {
	/**
	 * Called to validate an http request to see if it is upgradeable to a ws conn
	 */
	Validate(w http.ResponseWriter, r *http.Request) (C, error)

	/**
	 * Reads the next message from the ws conn.
	 */
	ReadMessage(ctx C, w *websocket.Conn) (I, error)

	/**
	 * Takes a message and format it to be written.
	 */
	WriteMessage(ctx C, conn *websocket.Conn, msg O, err error) error

	/**
	 * Called to send the next ping message.
	 */
	Pinger(ctx C) O

	/**
	 * Called to handle the next message from the input stream on the ws conn.
	 */
	HandleMessage(ctx C, msg I) error

	/**
	 * On created.
	 */
	OnStart(ctx C)

	/**
	 * Called to handle or suppress an error
	 */
	OnError(ctx C, err error) error

	/**
	 * Called when the connection closes.
	 */
	OnClose(ctx C)
	OnTimeout(ctx C) bool
}

type JSONHandler[C any] struct {
}

func (j *JSONHandler[C]) Validate(w http.ResponseWriter, r *http.Request) (out C, err error) {
	// All connections upgradeable
	return
}

/**
 * Reads the next message from the ws conn.
 */
func (j *JSONHandler[C]) ReadMessage(ctx C, conn *websocket.Conn) (out interface{}, err error) {
	err = conn.ReadJSON(&out)
	return
}

/**
 * Writes the next message to the ws conn.
 */
func (j *JSONHandler[C]) WriteMessage(ctx C, conn *websocket.Conn, msg interface{}, err error) error {
	if err == io.EOF {
		log.Println("Streamer closed...", err)
		// do nothing
		// SendJsonResponse(rw, nil, msg.Error)
		return nil
	} else if err != nil {
		return WSConnWriteError(conn, err)
	} else {
		return WSConnWriteMessage(conn, msg)
	}
}

/**
 * Called to send the next ping message.
 */
func (j *JSONHandler[C]) Pinger(ctx C) interface{} {
	return gut.StringMap{"type": "ping"}
}

/**
 * Called to handle the next message from the input stream on the ws conn.
 */
func (J *JSONHandler[C]) HandleMessage(ctx C, msg interface{}) error {
	return nil
}

func (J *JSONHandler[C]) OnStart(ctx C) {
}

func (J *JSONHandler[C]) OnError(ctx C, err error) error {
	return err
}

/**
 * Called when the connection closes.
 */
func (J *JSONHandler[C]) OnClose(ctx C) {
}

func (J *JSONHandler[C]) OnTimeout(ctx C) bool {
	return true
}

type WSHandlerConfig struct {
	Upgrader   websocket.Upgrader
	PingPeriod time.Duration
	PongPeriod time.Duration
}

func WithWSHandler[I any, O any, C any](config *WSHandlerConfig, h WSHandler[I, O, C]) http.HandlerFunc {
	if config == nil {
		config = &WSHandlerConfig{
			Upgrader: websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				CheckOrigin:     func(r *http.Request) bool { return true },
			},
			PingPeriod: time.Second * 30,
			PongPeriod: time.Second * 300,
		}
	}

	return func(rw http.ResponseWriter, req *http.Request) {
		ctx, err := h.Validate(rw, req)
		if err != nil {
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
		reader := conc.NewReader[I](func() (I, error) {
			return h.ReadMessage(ctx, conn)
		})
		defer reader.Stop()

		writer := conc.NewWriter[O](func(d O) error {
			h.WriteMessage(ctx, conn, d, nil)
			return nil
		})
		// All the core hapens here
		defer writer.Stop()

		lastReadAt := time.Now()
		pingTimer := time.NewTicker(config.PingPeriod)
		pongChecker := time.NewTicker(config.PongPeriod)
		defer pingTimer.Stop()
		defer pongChecker.Stop()

		sendPing := func() {
			if h.Pinger != nil {
				pingmsg := h.Pinger(ctx)
				writer.Write(pingmsg)
			}
		}

		stopChan := make(chan bool)
		defer close(stopChan)

		defer h.OnClose(ctx)

		sendPing()
		for {
			select {
			case <-stopChan:
				log.Println("Connection stopped....")
				return
			case <-pingTimer.C:
				sendPing()
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
				} else if h.HandleMessage != nil {
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
}
