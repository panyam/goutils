package conc

type EventType interface{}

/**
 * In our hub the main problem is given a message, identifying
 * all clients it should be sent to.  Instead of having a fixed
 * key -> HubClient[] mapping, it is easier to simply for each
 * message use a router delegate to identify all HubClients that
 * are candidates.  This Router does that.
 */
type Router[M any] interface {
	// Called when a new client has joined the group
	Add(client *HubClient[M]) error

	// Called when a client has dropped off
	Remove(client *HubClient[M]) error

	AddRoute(client *HubClient[M], eventKeys ...EventType) error
	RemoveRoute(client *HubClient[M], eventKeys ...EventType) error
	RouteMessage(msg M, err error, source *HubClient[M]) error
}

/**
 * A type of router that maintains an eventKey -> Conn[] by
 * a fixed event keys.
 */
type KVRouter[M any] struct {
	keyToClients map[EventType][]*HubClient[M]
	msgToKey     func(msg M) EventType
}

func NewKVRouter[M any](msgToKey func(msg M) EventType) *KVRouter[M] {
	return &KVRouter[M]{
		msgToKey:     msgToKey,
		keyToClients: make(map[EventType][]*HubClient[M]),
	}
}

func (r *KVRouter[M]) RouteMessage(msg M, e error, source *HubClient[M]) error {
	key := r.msgToKey(msg)
	for _, client := range r.keyToClients[key] {
		if source != client {
			if err := client.WriteMessage(msg, e); err != nil {
				// Should we fail - do so for now
				return err
			}
		}
	}
	return nil
}

func (r *KVRouter[M]) AddRoute(client *HubClient[M], eventKeys ...EventType) error {
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

func (r *KVRouter[M]) RemoveRoute(client *HubClient[M], eventKeys ...EventType) error {
	for _, eventKey := range eventKeys {
		r.removeClientFrom(eventKey, client)
	}
	return nil
}

func (r *KVRouter[M]) Add(client *HubClient[M]) error {
	// Does nothing
	return nil
}

func (r *KVRouter[M]) Remove(client *HubClient[M]) error {
	for k, _ := range r.keyToClients {
		r.removeClientFrom(k, client)
	}
	return nil
}

func (r *KVRouter[M]) removeClientFrom(eventKey EventType, client *HubClient[M]) {
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

/**
 * A type of router that forwards a message to *all* its
 * connected clients (except the one it originated from).
 */
type Broadcaster[M any] struct {
	allClients map[*HubClient[M]]bool
}

func NewBroadcaster[M any]() *Broadcaster[M] {
	return &Broadcaster[M]{
		allClients: make(map[*HubClient[M]]bool),
	}
}

func (r *Broadcaster[M]) RouteMessage(msg M, e error, source *HubClient[M]) error {
	for client, _ := range r.allClients {
		if source != client {
			if err := client.WriteMessage(msg, e); err != nil {
				// Should we fail - do so for now
				return err
			}
		}
	}
	return nil
}

func (r *Broadcaster[M]) AddRoute(client *HubClient[M], eventKeys ...EventType) error {
	// Does nothign since everything gets forwarded everywhere
	return nil
}

func (r *Broadcaster[M]) RemoveRoute(client *HubClient[M], eventKeys ...EventType) error {
	// Does nothign since everything gets forwarded everywhere
	return nil
}

func (r *Broadcaster[M]) Add(client *HubClient[M]) error {
	// Does nothing
	r.allClients[client] = true
	return nil
}

func (r *Broadcaster[M]) Remove(client *HubClient[M]) error {
	if _, ok := r.allClients[client]; ok {
		delete(r.allClients, client)
	}
	return nil
}
