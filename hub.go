package conc

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type HubWriter[M any] func(msg M, err error) error

type HubClient[M any] struct {
	hub          *Hub[M]
	id           string
	WriteMessage HubWriter[M]
}

func (h *HubClient[M]) GetId() string {
	return h.id
}

type HubControlEvent[M any] struct {
	Client               *HubClient[M]
	Quit                 bool
	Pause                bool
	AddedSubscriptions   []EventType
	RemovedSubscriptions []EventType
}

type HubMessage[M any] struct {
	Message  M
	Error    error
	Callback chan M
}

type Hub[M any] struct {
	idCounter uint64

	routerLock     sync.RWMutex
	router         Router[M]
	controlChannel chan *HubControlEvent[M]
	newMsgChannel  chan HubMessage[M]
	stopChannel    chan bool
}

func NewHub[M any](router Router[M]) *Hub[M] {
	if router == nil {
		router = NewBroadcaster[M]()
	}
	out := &Hub[M]{
		router:         router,
		controlChannel: make(chan *HubControlEvent[M]),
		newMsgChannel:  make(chan HubMessage[M]),
		stopChannel:    make(chan bool),
	}
	go out.start()
	return out
}

func (h *Hub[M]) Connect(writer HubWriter[M]) *HubClient[M] {
	// Pause till it is closed - in the mean time the publisher will
	// be sending messages with queued up events on this channel
	hc := HubClient[M]{
		hub:          h,
		id:           fmt.Sprintf("%d", h.idCounter),
		WriteMessage: writer,
	}
	h.idCounter++
	h.routerLock.Lock()
	defer h.routerLock.Unlock()
	h.router.Add(&hc)
	return &hc
}

func (s *Hub[M]) start() error {
	// 1. First start a stream with a reader so we can read sub/unsub
	// messages on this
	log.Println("Starting Hub...")
	ticker := time.NewTicker(1 * time.Second)

	defer func() {
		log.Println("Stopping Hub")
		close(s.controlChannel)
		close(s.newMsgChannel)
		close(s.stopChannel)
		// TODO - relinquish ownership of the router
		// s.router.Close()
	}()
	for {
		select {
		case <-ticker.C:
			// Check if things like connections have dropped off etc
			break
		case <-s.stopChannel:
			return nil
		case evt := <-s.controlChannel:
			// handle subscriptions, unsubs and even heart beats here
			if evt.Quit {
				// A connection is quitting so remove it form all routes
				s.router.Remove(evt.Client)
			} else {
				s.router.AddRoute(evt.Client, evt.AddedSubscriptions...)
				s.router.RemoveRoute(evt.Client, evt.RemovedSubscriptions...)
			}
			break
		case msg := <-s.newMsgChannel:
			// Handle fanout here
			// TODO - how can we add error and source here?
			s.router.RouteMessage(msg.Message, nil, nil)
			if msg.Callback != nil {
				msg.Callback <- msg.Message
			}
			break
		}
	}
}

func (h *HubClient[M]) Disconnect() {
	h.hub.controlChannel <- &HubControlEvent[M]{
		Client: h,
		Quit:   true,
	}
}

func (h *HubClient[M]) Subscribe(eventTypes ...EventType) {
	h.hub.controlChannel <- &HubControlEvent[M]{
		Client:             h,
		AddedSubscriptions: eventTypes,
	}
}

func (h *HubClient[M]) Unsubscribe(eventTypes ...EventType) {
	h.hub.controlChannel <- &HubControlEvent[M]{
		Client:               h,
		RemovedSubscriptions: eventTypes,
	}
}

func (s *Hub[M]) Send(message M, err error, callbackChan chan M) error {
	s.newMsgChannel <- HubMessage[M]{
		Message:  message,
		Error:    err,
		Callback: callbackChan,
	}
	return nil
}

/**
 * Stop the Hub and cleanup.
 */
func (s *Hub[M]) Stop() {
	s.stopChannel <- true
}
