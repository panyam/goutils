package ws

import (
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

type WSMsgType gut.StringMap
type WSFanOut = conc.FanOut[conc.Message[WSMsgType], conc.Message[WSMsgType]]

type WSMux struct {
	Upgrader            websocket.Upgrader
	HandleClientMessage func(ws *WSMux, msg WSMsgType) error
	allClients          map[*WSClient]map[string]bool
	clientsByTopic      map[string]*WSFanOut
	connsLock           sync.RWMutex
}

func NewWSMux() *WSMux {
	return &WSMux{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		allClients:     make(map[*WSClient]map[string]bool),
		clientsByTopic: make(map[string]*WSFanOut),
	}
}

// Publishes a message to all clients on caring about given topic
func (w *WSMux) Publish(topicId string, msg WSMsgType) error {
	w.connsLock.RLock()
	defer w.connsLock.RUnlock()
	if fanout, ok := w.clientsByTopic[topicId]; ok && fanout != nil {
		fanout.Send(conc.Message[WSMsgType]{Value: msg})
	}
	return nil
}

// Sets the "listening" client on a topic. This ensures that there is
// only a single client for the given topic.
func (w *WSMux) SetForTopic(topicId string, client *WSClient) error {
	w.connsLock.Lock()
	defer w.connsLock.Unlock()

	w.allClients[client] = map[string]bool{topicId: true}
	return nil
}

// Adds a client to a particular topic id.  This makes it eligible to
// receive messages sent on a particular topic
// Normally this is called by
func (w *WSMux) AddToTopic(topicId string, client *WSClient) error {
	w.connsLock.Lock()
	defer w.connsLock.Unlock()
	return w.addToTopic(topicId, client)
}

// Remove a client from particular topic id.  This makes it stop receiving
// messages sent on the particular topic
func (w *WSMux) RemoveFromTopic(topicId string, client *WSClient) error {
	w.connsLock.Lock()
	defer w.connsLock.Unlock()
	return w.removeFromTopic(topicId, client)
}

func (w *WSMux) RemoveClient(client *WSClient) error {
	w.connsLock.Lock()
	defer w.connsLock.Unlock()
	return w.removeClient(client)
}

func (w *WSMux) addToTopic(topicId string, client *WSClient) error {
	if topicSet, ok := w.clientsByTopic[topicId]; !ok || topicSet == nil {
		w.clientsByTopic[topicId] = conc.NewIDFanOut[conc.Message[WSMsgType]](nil, nil)
	}
	if clientSet, ok := w.allClients[client]; !ok || clientSet == nil {
		w.allClients[client] = make(map[string]bool)
	}
	w.allClients[client][topicId] = true
	w.clientsByTopic[topicId].Add(client.writer.SendChan(), nil)
	return nil
}

func (w *WSMux) removeFromTopic(topicId string, client *WSClient) error {
	if fanout, ok := w.clientsByTopic[topicId]; ok && fanout != nil {
		fanout.Remove(client.writer.SendChan())
	}
	if clientSet, ok := w.allClients[client]; !ok || clientSet == nil {
		delete(clientSet, topicId)
	}
	return nil
}

func (w *WSMux) removeClient(client *WSClient) error {
	for topicId, _ := range w.allClients[client] {
		w.removeFromTopic(topicId, client)
	}
	return nil
}

/**
 * Called when a new connection arrives.
 */
func (w *WSMux) Subscribe(req *http.Request, writer http.ResponseWriter) (*WSClient, error) {
	wsConn, err := w.Upgrader.Upgrade(writer, req, nil)
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v", err)
		SendJsonResponse(writer, nil, err)
		return nil, err
	}
	defer log.Println("Closing WS connection from Agent FE")
	defer wsConn.Close()
	out := WSClient{
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

type WSClient struct {
	PingPeriod time.Duration
	PongPeriod time.Duration
	wsMux      *WSMux
	wsConn     *websocket.Conn
	reader     *conc.Reader[WSMsgType]
	writer     *conc.Writer[conc.Message[WSMsgType]]
	stopChan   chan bool
}

func (w *WSClient) Start() error {
	lastPingedAt := time.Now()
	pingChecker := time.NewTicker(time.Second)
	w.stopChan = make(chan bool)
	defer pingChecker.Stop()
	defer w.cleanup()
	defer log.Println("Client exiting....")

	w.wsConn.SetReadDeadline(time.Now().Add(w.PongPeriod))

	for {
		select {
		case <-w.stopChan:
			log.Println("Client stopped....")
			return nil
		case <-pingChecker.C:
			if time.Now().Sub(lastPingedAt) > w.PongPeriod {
				// Lost connection with client so can drop off?
				log.Println("Connect stopped pinging. Killing proxy connection...")
				return nil
			}
			break
		case result := <-w.reader.RecvChan():
			lastPingedAt = time.Now()
			w.wsConn.SetReadDeadline(time.Now().Add(w.PongPeriod))
			if result.Error != nil {
				log.Println("WebSocket Error: ", result.Error)
				return result.Error
			} else {
				// we have an actual message being sent on this channel - typically
				// dont need to do anything as we are using these for outbound connections
				// only to write to a listening agent FE so can just log and drop any
				// thing sent by agent FE here - this can change later
				log.Println("Received message from client: ", result.Value)
				w.wsMux.HandleClientMessage(w.wsMux, result.Value)
			}
			break
		}
	}
}

func (w *WSClient) Stop() {
	defer log.Println("Client stop issued...")
	w.stopChan <- true
	defer log.Println("Client accepted issued.")
}

func (w *WSClient) cleanup() {
	defer log.Println("Cleaning up client....")
	defer w.wsMux.RemoveClient(w)
	w.reader.Stop()
	w.writer.Stop()
	close(w.stopChan)
	defer log.Println("Finished cleaning up client.")
}
