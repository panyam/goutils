package conc

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

	AddRoute(client *HubClient[M], topicIds ...TopicIdType) error
	RemoveRoute(client *HubClient[M], topicIds ...TopicIdType) error
	RouteMessage(msg Message[M], source *HubClient[M]) error
}

/**
 * A type of router that maintains an topicId -> Conn[] by
 * a fixed topics.
 */
type KVRouter[M any] struct {
	keyToClients map[TopicIdType][]*HubClient[M]
	msgToKey     func(msg M) TopicIdType
}

func NewKVRouter[M any](msgToKey func(msg M) TopicIdType) *KVRouter[M] {
	return &KVRouter[M]{
		msgToKey:     msgToKey,
		keyToClients: make(map[TopicIdType][]*HubClient[M]),
	}
}

func (r *KVRouter[M]) RouteMessage(msg Message[M], source *HubClient[M]) error {
	key := r.msgToKey(msg.Value)
	for _, client := range r.keyToClients[key] {
		if source != client {
			if err := client.Write(msg); err != nil {
				// Should we fail - do so for now
				return err
			}
		}
	}
	return nil
}

func (r *KVRouter[M]) AddRoute(client *HubClient[M], topicIds ...TopicIdType) error {
	for _, topicId := range topicIds {
		if clients, ok := r.keyToClients[topicId]; ok {
			for _, c := range clients {
				if c.GetId() == client.GetId() {
					// Already exists
					break
				}
			}
		}
		r.keyToClients[topicId] = append(r.keyToClients[topicId], client)
	}
	return nil
}

func (r *KVRouter[M]) RemoveRoute(client *HubClient[M], topicIds ...TopicIdType) error {
	for _, topicId := range topicIds {
		r.removeClientFrom(topicId, client)
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

func (r *KVRouter[M]) removeClientFrom(topicId TopicIdType, client *HubClient[M]) {
	if clients, ok := r.keyToClients[topicId]; ok {
		for i, c := range clients {
			if c.GetId() == client.GetId() {
				// Found it - remove and exit (order not import so just replace
				// with last and trim
				clients[i] = clients[len(clients)-1]
				clients = clients[:len(clients)-1]
				r.keyToClients[topicId] = clients
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

func (r *Broadcaster[M]) RouteMessage(msg Message[M], source *HubClient[M]) error {
	for client, _ := range r.allClients {
		if source != client {
			if err := client.Write(msg); err != nil {
				// Should we fail - do so for now
				return err
			}
		}
	}
	return nil
}

func (r *Broadcaster[M]) AddRoute(client *HubClient[M], topicIds ...TopicIdType) error {
	// Does nothign since everything gets forwarded everywhere
	return nil
}

func (r *Broadcaster[M]) RemoveRoute(client *HubClient[M], topicIds ...TopicIdType) error {
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
