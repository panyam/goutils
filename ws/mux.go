package ws

import (
	"log"
	"sync"

	"github.com/panyam/goutils/conc"
)

type WSMux[I any, O any] struct {
	allConns     map[*WSConn[I, O]]map[string]bool
	connsByTopic map[string]*conc.FanOut[conc.Message[O], conc.Message[O]]
	connsLock    sync.RWMutex
	connIdLock   sync.RWMutex
	connIdMap    map[string]bool
}

func NewWSMux[I any, O any]() *WSMux[I, O] {
	return &WSMux[I, O]{
		allConns:     make(map[*WSConn[I, O]]map[string]bool),
		connsByTopic: make(map[string]*conc.FanOut[conc.Message[O], conc.Message[O]]),
		connIdMap:    make(map[string]bool),
	}
}

func (w *WSMux[I, O]) Update(actions func(w *WSMux[I, O]) error) error {
	log.Println("Acquiring update lock")
	defer w.connsLock.Unlock()
	defer log.Println("Releasing update lock")
	w.connsLock.Lock()
	return actions(w)
}

func (w *WSMux[I, O]) View(actions func(w *WSMux[I, O]) error) error {
	log.Println("Acquiring view lock")
	defer w.connsLock.RUnlock()
	defer log.Println("Releasing view lock")
	w.connsLock.RLock()
	return actions(w)
}

// Publishes a message to all conns on caring about given topic
func (w *WSMux[I, O]) Publish(topicId string, msg O, lock bool) error {
	if lock {
		w.connsLock.RLock()
		defer w.connsLock.RUnlock()
	}
	if fanout, ok := w.connsByTopic[topicId]; ok && fanout != nil {
		fanout.Send(conc.Message[O]{Value: msg})
	}
	return nil
}

func (w *WSMux[I, O]) GetConnsForTopic(topicId string, lock bool) (out []*WSConn[I, O]) {
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
func (w *WSMux[I, O]) AddToTopic(topicId string, conn *WSConn[I, O], lock bool) error {
	if lock {
		w.connsLock.Lock()
		defer w.connsLock.Unlock()
	}
	if topicSet, ok := w.connsByTopic[topicId]; !ok || topicSet == nil {
		w.connsByTopic[topicId] = conc.NewIDFanOut[conc.Message[O]](nil, nil)
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
func (w *WSMux[I, O]) RemoveFromTopic(topicId string, conn *WSConn[I, O], lock bool) error {
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

func (w *WSMux[I, O]) RemoveConn(conn *WSConn[I, O], lock bool) error {
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
