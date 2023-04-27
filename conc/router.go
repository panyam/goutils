package conc

import (
	"sync"
)

/**
 * In our streamer the main problem is given a message, identifying
 * all connections it should be sent to.  Instead of having a fixed
 * key -> HubClient[] mapping, it is easier to simply for each
 * message use a router delegate to identify all HubClients that
 * are candidates.  This Router does that.
 */
type Router[M any] interface {
	AddRoute(client *HubClient[M], eventKeys ...string) error
	RemoveRoute(client *HubClient[M], eventKeys ...string) error
	Remove(client *HubClient[M]) error
	RouteMessage(eventKey string, msg M) error
}

/**
 * A type of router that maintains an eventKey -> Conn[] by
 * a fixed event keys.
 */
type KVRouter[M any] struct {
	keyToClients map[string][]*HubClient[M]
	rwlock       sync.RWMutex
}

func NewKVRouter[M any]() *KVRouter[M] {
	return &KVRouter[M]{
		keyToClients: make(map[string][]*HubClient[M]),
	}
}

func (r *KVRouter[M]) RouteMessage(eventKey string, msg M) error {
	r.rwlock.RLock()
	defer r.rwlock.RUnlock()
	for _, client := range r.keyToClients[eventKey] {
		if err := client.WriteMessage(msg); err != nil {
			// Should we fail - do so for now
			return err
		}
	}
	return nil
}

func (r *KVRouter[M]) AddRoute(client *HubClient[M], eventKeys ...string) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()
	for _, eventKey := range eventKeys {
		if clients, ok := r.keyToClients[eventKey]; ok {
			for _, c := range clients {
				if c.GetId() == client.GetId() {
					// Already exists
					break
				}
			}
		}
		r.keyToClients[eventKey] = append(r.keyToClients[eventKey], client)
	}
	return nil
}

func (r *KVRouter[M]) RemoveRoute(client *HubClient[M], eventKeys ...string) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()

	for _, eventKey := range eventKeys {
		r.removeClientFrom(eventKey, client)
	}
	return nil
}

func (r *KVRouter[M]) Remove(client *HubClient[M]) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()
	for k, _ := range r.keyToClients {
		r.removeClientFrom(k, client)
	}
	return nil
}

func (r *KVRouter[M]) removeClientFrom(eventKey string, client *HubClient[M]) {
	if clients, ok := r.keyToClients[eventKey]; ok {
		for i, c := range clients {
			if c.GetId() == client.GetId() {
				// Found it - remove and exit (order not import so just replace
				// with last and trim
				clients[i] = clients[len(clients)-1]
				r.keyToClients[eventKey] = clients[:len(clients)-1]
				break
			}
		}
	}
}
