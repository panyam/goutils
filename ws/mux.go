package ws

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gut "github.com/panyam/goutils/utils"
)

type WSMsgType interface{}
type WSFanOut = conc.FanOut[conc.Message[WSMsgType], conc.Message[WSMsgType]]

type WSMux struct {
	Upgrader     websocket.Upgrader
	allConns     map[*WSConn]map[string]bool
	connsByTopic map[string]*WSFanOut
	connsLock    sync.RWMutex
	connIdLock   sync.RWMutex
	connIdMap    map[string]bool
}

func NewWSMux() *WSMux {
	return &WSMux{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		allConns:     make(map[*WSConn]map[string]bool),
		connsByTopic: make(map[string]*WSFanOut),
		connIdMap:    make(map[string]bool),
	}
}

func (w *WSMux) Update(actions func(w *WSMux) error) error {
	log.Println("Acquiring update lock")
	defer w.connsLock.Unlock()
	defer log.Println("Releasing update lock")
	w.connsLock.Lock()
	return actions(w)
}

func (w *WSMux) View(actions func(w *WSMux) error) error {
	log.Println("Acquiring view lock")
	defer w.connsLock.RUnlock()
	defer log.Println("Releasing view lock")
	w.connsLock.RLock()
	return actions(w)
}

// Publishes a message to all conns on caring about given topic
func (w *WSMux) Publish(topicId string, msg WSMsgType, lock bool) error {
	if lock {
		w.connsLock.RLock()
		defer w.connsLock.RUnlock()
	}
	if fanout, ok := w.connsByTopic[topicId]; ok && fanout != nil {
		fanout.Send(conc.Message[WSMsgType]{Value: msg})
	}
	return nil
}

func (w *WSMux) GetConnsForTopic(topicId string, lock bool) (out []*WSConn) {
	if lock {
		w.connsLock.RLock()
		defer w.connsLock.RUnlock()
	}
	for conn, topicset := range w.allConns {
		if exists, ok := topicset[topicId]; ok && exists {
			out = append(out, conn)
			break
		}
	}
	return
}

// Adds a conn to a particular topic id.  This makes it eligible to
// receive messages sent on a particular topic
// Normally this is called by
func (w *WSMux) AddToTopic(topicId string, conn *WSConn, lock bool) error {
	if lock {
		w.connsLock.Lock()
		defer w.connsLock.Unlock()
	}
	if topicSet, ok := w.connsByTopic[topicId]; !ok || topicSet == nil {
		w.connsByTopic[topicId] = conc.NewIDFanOut[conc.Message[WSMsgType]](nil, nil)
	}
	if connSet, ok := w.allConns[conn]; !ok || connSet == nil {
		w.allConns[conn] = make(map[string]bool)
	}
	w.allConns[conn][topicId] = true
	w.connsByTopic[topicId].Add(conn.writer.SendChan(), nil)
	return nil
}

// Remove a conn from particular topic id.  This makes it stop receiving
// messages sent on the particular topic
func (w *WSMux) RemoveFromTopic(topicId string, conn *WSConn, lock bool) error {
	if lock {
		w.connsLock.Lock()
		defer w.connsLock.Unlock()
	}
	if fanout, ok := w.connsByTopic[topicId]; ok && fanout != nil {
		fanout.Remove(conn.writer.SendChan(), nil)
	}
	if connSet, ok := w.allConns[conn]; ok && connSet != nil {
		delete(connSet, topicId)
	}
	return nil
}

func (w *WSMux) RemoveConn(conn *WSConn, lock bool) error {
	if lock {
		w.connsLock.Lock()
		defer w.connsLock.Unlock()
	}
	log.Println("Before Removing Conn: ", conn, len(w.allConns), w.allConns, w.allConns[conn])
	for topicId := range w.allConns[conn] {
		w.RemoveFromTopic(topicId, conn, false)
	}
	delete(w.allConns, conn)
	log.Println("After Removing Conn: ", len(w.allConns), w.allConns, w.allConns[conn])
	return nil
}

func (w *WSMux) lockConnId(connId string) error {
	w.connIdLock.Lock()
	defer w.connIdLock.Unlock()
	if _, ok := w.connIdMap[connId]; !ok {
		return errors.New("id already taken")
	}
	w.connIdMap[connId] = true
	return nil
}

func (w *WSMux) nextConnId() string {
	w.connIdLock.Lock()
	defer w.connIdLock.Unlock()
	for {
		connId := gut.RandString(10, "")
		if _, ok := w.connIdMap[connId]; !ok {
			// found it
			w.connIdMap[connId] = true
			return connId
		}
	}
}

/**
 * Called when a new connection arrives.
 */
func (w *WSMux) Subscribe(req *http.Request, rw http.ResponseWriter, connId string) (*WSConn, error) {
	if connId == "" {
		// autogen one
		connId = w.nextConnId()
	} else if err := w.lockConnId(connId); err != nil {
		return nil, err
	}
	wsConn, err := w.Upgrader.Upgrade(rw, req, nil)
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v", err)
		SendJsonResponse(rw, nil, err)
		return nil, err
	}
	out := NewWSConn(wsConn, nil, nil)
	out.ConnId = connId
	out.wsMux = w
	return out, nil
}

type WSConn struct {
	ConnId      string
	LastReadAt  time.Time
	PingPeriod  time.Duration
	PongPeriod  time.Duration
	pingTimer   *time.Ticker
	pongChecker *time.Ticker
	wsMux       *WSMux
	wsConn      *websocket.Conn
	reader      *conc.Reader[WSMsgType]
	writer      *conc.Writer[conc.Message[WSMsgType]]
	stopChan    chan bool

	// Callback to decide what the ping message should be
	Pinger func(*WSConn) (WSMsgType, error)

	// Called when read has timed but giving the client
	// a chance to override the timeout
	OnReadTimeout func(*WSConn) bool

	// Called before the connection is closed so the client
	// can perform any clietn before the internals are torn down
	OnClose func(*WSConn)

	// Called when a new message is available to be handled
	HandleMessage func(w *WSConn, msg WSMsgType) error
}

func NewWSConn(conn *websocket.Conn, reader *conc.Reader[WSMsgType], writer *conc.Writer[conc.Message[WSMsgType]]) (w *WSConn) {
	if reader == nil {
		reader = conc.NewReader(func() (out WSMsgType, err error) {
			err = conn.ReadJSON(&out)
			return
		})
	}
	if writer == nil {
		writer = conc.NewWriter(func(msg conc.Message[WSMsgType]) error {
			if msg.Error == io.EOF {
				log.Println("Streamer closed...", msg.Error)
				// do nothing
				// SendJsonResponse(rw, nil, msg.Error)
				return msg.Error
			} else if msg.Error != nil {
				return WSConnWriteError(conn, msg.Error)
			} else {
				return WSConnWriteMessage(conn, msg.Value)
			}
		})
	}
	return &WSConn{
		PingPeriod: 10 * time.Second,
		PongPeriod: 60 * time.Second,
		wsConn:     conn,
		reader:     reader,
		writer:     writer,
	}
}

func (w *WSConn) sendPing() {
	if w.Pinger != nil {
		if pingmsg, err := w.Pinger(w); err != nil {
			log.Println("Ping failed: ", err)
		} else if pingmsg != nil {
			w.Send(pingmsg)
		}
	}
}

func (w *WSConn) Start() error {
	w.LastReadAt = time.Now()
	w.pingTimer = time.NewTicker(w.PingPeriod)
	w.pongChecker = time.NewTicker(w.PongPeriod)
	w.stopChan = make(chan bool)
	defer w.cleanup()

	w.wsConn.SetReadDeadline(time.Now().Add(w.PongPeriod))

	w.sendPing()
	for {
		select {
		case <-w.stopChan:
			log.Println("Connection stopped....")
			return nil
		case <-w.pingTimer.C:
			w.sendPing()
			break
		case <-w.pongChecker.C:
			hb_delta := time.Now().Sub(w.LastReadAt).Seconds()
			if hb_delta > w.PongPeriod.Seconds() {
				// Lost connection with conn so can drop off?
				if w.OnReadTimeout == nil || w.OnReadTimeout(w) {
					log.Printf("Last heart beat more than %d seconds ago.  Killing connection", int(hb_delta))
					return nil
				}
			}
			break
		case result := <-w.reader.RecvChan():
			w.LastReadAt = time.Now()
			w.wsConn.SetReadDeadline(time.Now().Add(w.PongPeriod))
			if result.Error != nil {
				if result.Error != io.EOF {
					if ce, ok := result.Error.(*websocket.CloseError); ok {
						log.Println("WebSocket Closed: ", ce)
						switch ce.Code {
						case websocket.CloseAbnormalClosure:
						case websocket.CloseNormalClosure:
						case websocket.CloseGoingAway:
							return nil
						}
					} else {
						log.Println("Unknown Error: ", result.Error, io.EOF)
					}
					return result.Error
				}
			} else if w.HandleMessage != nil {
				// we have an actual message being sent on this channel - typically
				// dont need to do anything as we are using these for outbound connections
				// only to write to a listening agent FE so can just log and drop any
				// thing sent by agent FE here - this can change later
				w.HandleMessage(w, result.Value)
			}
			break
		}
	}
}

func (w *WSConn) Stop() {
	if w.stopChan != nil {
		log.Println("Stop issued for conn: ", w.ConnId)
		w.stopChan <- true
		log.Println("Conn stopped: ", w.ConnId)
	}
}

func (w *WSConn) Send(msg WSMsgType) {
	w.writer.Send(conc.Message[WSMsgType]{Value: msg})
}

func (w *WSConn) cleanup() {
	log.Println("Cleaning up conn....")
	defer log.Println("Finished cleaning up conn.")
	close(w.stopChan)
	w.stopChan = nil

	w.pingTimer.Stop()
	w.pingTimer = nil

	w.pongChecker.Stop()
	w.pongChecker = nil

	// Remove the connections first if this is a server side connection
	if w.wsMux != nil {
		w.wsMux.RemoveConn(w, true)
	}

	// Then stop reader/writer.  Order is important
	// as the conn is using the reader/writer
	w.reader.Stop()
	w.reader = nil

	w.writer.Stop()
	w.writer = nil

	w.wsConn.Close()
}
