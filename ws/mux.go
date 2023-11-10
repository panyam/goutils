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
	w.connsLock.Lock()
	defer w.connsLock.Unlock()
	defer log.Println("Releasing update lock")
	return actions(w)
}

func (w *WSMux) View(actions func(w *WSMux) error) error {
	log.Println("Acquiring view lock")
	w.connsLock.RLock()
	defer w.connsLock.RUnlock()
	defer log.Println("Releasing view lock")
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

// Sets the "listening" conn on a topic. This ensures that there is
// only a single conn for the given topic.
/*
func (w *WSMux) SetForTopic(topicId string, conn *WSConn) error {
	w.connsLock.Lock()
	defer w.connsLock.Unlock()

	// go through all existing conns and if that existing conn
	// is registered to receive on this topic then remove its subscription
	// (unless it is us)
	var toremove []*WSConn
	for existing := range w.allConns {
		if conn != existing {
			toremove = append(toremove, existing)
		} else {
			panic("Connection already exists")
		}
	}
	for _, conn := range toremove {
		w.removeFromTopic(topicId, conn)
		conn.Stop()
	}
	w.addToTopic(topicId, conn)
	return nil
}
*/

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
		fanout.Remove(conn.writer.SendChan())
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
	log.Println("Before Removing: ", w.allConns)
	for topicId := range w.allConns[conn] {
		w.RemoveFromTopic(topicId, conn, false)
	}
	log.Println("After Removing: ", w.allConns)
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
func (w *WSMux) Subscribe(req *http.Request, writer http.ResponseWriter, connId string) (*WSConn, error) {
	if connId == "" {
		// autogen one
		connId = w.nextConnId()
	} else if err := w.lockConnId(connId); err != nil {
		return nil, err
	}
	wsConn, err := w.Upgrader.Upgrade(writer, req, nil)
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v", err)
		SendJsonResponse(writer, nil, err)
		return nil, err
	}
	out := WSConn{
		ConnId:     connId,
		PingPeriod: 10 * time.Second,
		PongPeriod: 60 * time.Second,
		wsMux:      w,
		wsConn:     wsConn,
		reader: conc.NewReader(func() (WSMsgType, error) {
			var out WSMsgType
			err := wsConn.ReadJSON(&out)
			return out, err
		}),
		writer: conc.NewWriter(func(msg conc.Message[WSMsgType]) error {
			if msg.Error == io.EOF {
				log.Println("Streamer closed...", msg.Error)
				SendJsonResponse(writer, nil, msg.Error)
				return msg.Error
			} else if msg.Error != nil {
				return WSConnWriteError(wsConn, msg.Error)
			} else {
				return WSConnWriteMessage(wsConn, msg.Value)
			}
		}),
	}
	return &out, nil
}

type WSConn struct {
	ConnId        string
	PingPeriod    time.Duration
	PongPeriod    time.Duration
	pingTimer     *time.Ticker
	pongChecker   *time.Ticker
	wsMux         *WSMux
	wsConn        *websocket.Conn
	reader        *conc.Reader[WSMsgType]
	writer        *conc.Writer[conc.Message[WSMsgType]]
	stopChan      chan bool
	Pinger        func(*WSConn) (WSMsgType, error)
	OnReadTimeout func(*WSConn) bool
	LastReadAt    time.Time
	HandleMessage func(w *WSConn, msg WSMsgType) error
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
			if time.Now().Sub(w.LastReadAt).Seconds() > w.PongPeriod.Seconds() {
				// Lost connection with conn so can drop off?
				if w.OnReadTimeout == nil || w.OnReadTimeout(w) {
					log.Println("Connect stopped pinging. Killing proxy connection...")
					return nil
				}
			}
			break
		case result := <-w.reader.RecvChan():
			w.LastReadAt = time.Now()
			w.wsConn.SetReadDeadline(time.Now().Add(w.PongPeriod))
			if result.Error != nil {
				if result.Error != io.EOF {
					log.Println("WebSocket Error: ", result.Error, io.EOF)
					return result.Error
				}
			} else {
				// we have an actual message being sent on this channel - typically
				// dont need to do anything as we are using these for outbound connections
				// only to write to a listening agent FE so can just log and drop any
				// thing sent by agent FE here - this can change later
				log.Println("Received message from conn: ", result.Value)
				if w.HandleMessage != nil {
					w.HandleMessage(w, result.Value)
				}
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
	w.wsMux.RemoveConn(w, true)
	defer log.Println("Finished cleaning up conn.")
	w.pingTimer.Stop()
	w.pongChecker.Stop()
	w.reader.Stop()
	w.writer.Stop()
	w.wsConn.Close()
	close(w.stopChan)
	w.stopChan = nil
	w.pingTimer = nil
	w.pongChecker = nil
	w.reader = nil
	w.writer = nil
}
