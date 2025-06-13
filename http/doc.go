/*
Package http provides utilities for HTTP request handling and production-grade WebSocket connections.

This package contains two main areas of functionality:

 1. HTTP utilities for simplified request/response handling
 2. WebSocket abstractions built on Gorilla WebSocket for real-time applications

# WebSocket Framework

The WebSocket framework provides a production-ready abstraction over Gorilla WebSocket
with automatic connection management, heartbeat detection, and lifecycle hooks.

## Key Features

  - Production-grade connection management with automatic heartbeat detection
  - Lifecycle hooks for connection start, close, timeout, and error handling
  - Thread-safe message broadcasting to multiple clients
  - Automatic ping-pong mechanism to prevent connection timeouts
  - Type-safe message handling with Go generics
  - Built-in JSON message support with JSONConn
  - Configurable timeouts and intervals for different deployment scenarios

## Quick Start

The simplest way to create a WebSocket endpoint is using the built-in JSONConn:

	type EchoConn struct {
		gohttp.JSONConn
	}

	func (e *EchoConn) HandleMessage(msg any) error {
		// Echo the message back to the client
		e.Writer.Send(conc.Message[any]{Value: msg})
		return nil
	}

	type EchoHandler struct{}

	func (h *EchoHandler) Validate(w http.ResponseWriter, r *http.Request) (*EchoConn, bool) {
		return &EchoConn{}, true // Accept all connections
	}

	// Register with HTTP router
	router.HandleFunc("/echo", gohttp.WSServe(&EchoHandler{}, nil))

## Architecture

The framework uses three main interfaces:

### WSConn[I any]
Represents a WebSocket connection that can handle typed messages:

	type WSConn[I any] interface {
		BiDirStreamConn[I]
		ReadMessage(w *websocket.Conn) (I, error)
		OnStart(conn *websocket.Conn) error
	}

### WSHandler[I any, S WSConn[I]]
Validates HTTP requests and creates WebSocket connections:

	type WSHandler[I any, S WSConn[I]] interface {
		Validate(w http.ResponseWriter, r *http.Request) (S, bool)
	}

### BiDirStreamConn[I any]
Provides lifecycle and message handling methods:

	type BiDirStreamConn[I any] interface {
		SendPing() error
		Name() string
		ConnId() string
		HandleMessage(msg I) error
		OnError(err error) error
		OnClose()
		OnTimeout() bool
	}

## Advanced Usage

### Multi-user Chat Server

	type ChatServer struct {
		clients map[string]*ChatConn
		rooms   map[string]*Room
		mu      sync.RWMutex
	}

	type ChatConn struct {
		gohttp.JSONConn
		server   *ChatServer
		username string
		roomName string
	}

	func (c *ChatConn) OnStart(conn *websocket.Conn) error {
		if err := c.JSONConn.OnStart(conn); err != nil {
			return err
		}

		// Register with server
		c.server.mu.Lock()
		c.server.clients[c.ConnId()] = c
		c.server.mu.Unlock()

		return nil
	}

	func (c *ChatConn) HandleMessage(msg any) error {
		msgMap := msg.(map[string]any)

		switch msgMap["type"].(string) {
		case "chat":
			// Broadcast to all clients in room
			c.server.broadcastToRoom(c.roomName, msgMap)
		case "join_room":
			// Switch rooms
			c.joinRoom(msgMap["room"].(string))
		}

		return nil
	}

### Authentication and Authorization

	type SecureConn struct {
		gohttp.JSONConn
		userID string
		roles  []string
	}

	func (s *SecureConn) HandleMessage(msg any) error {
		msgMap := msg.(map[string]any)
		messageType := msgMap["type"].(string)

		if !s.hasPermission(messageType) {
			errorMsg := map[string]any{
				"type":  "error",
				"error": "Insufficient permissions",
			}
			s.Writer.Send(conc.Message[any]{Value: errorMsg})
			return nil
		}

		// Process authorized message
		return s.processMessage(msgMap)
	}

	type AuthHandler struct{}

	func (h *AuthHandler) Validate(w http.ResponseWriter, r *http.Request) (*SecureConn, bool) {
		token := r.Header.Get("Authorization")

		// Validate JWT token
		claims, err := validateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return nil, false
		}

		return &SecureConn{
			userID: claims["userID"].(string),
			roles:  claims["roles"].([]string),
		}, true
	}

### Custom Ping-Pong with Latency Tracking

	type MonitoredConn struct {
		gohttp.JSONConn
		lastPingTime time.Time
		pingCount    int64
	}

	func (m *MonitoredConn) SendPing() error {
		m.pingCount++
		m.lastPingTime = time.Now()

		pingMsg := map[string]any{
			"type":      "ping",
			"pingId":    m.pingCount,
			"timestamp": m.lastPingTime.Unix(),
		}

		m.Writer.Send(conc.Message[any]{Value: pingMsg})
		return nil
	}

	func (m *MonitoredConn) HandleMessage(msg any) error {
		msgMap := msg.(map[string]any)

		if msgMap["type"].(string) == "pong" {
			latency := time.Since(m.lastPingTime)
			log.Printf("Ping-pong latency: %v", latency)
		}

		return nil
	}

## Configuration

### Production Configuration

	config := &gohttp.WSConnConfig{
		BiDirStreamConfig: &gohttp.BiDirStreamConfig{
			PingPeriod: time.Second * 30,  // Send ping every 30 seconds
			PongPeriod: time.Second * 300, // Timeout after 5 minutes
		},
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// Implement proper origin checking
				return isValidOrigin(r.Header.Get("Origin"))
			},
		},
	}

	router.HandleFunc("/ws", gohttp.WSServe(handler, config))

### Development Configuration

	config := gohttp.DefaultWSConnConfig()
	config.PingPeriod = time.Second * 10  // Faster pings for development
	config.PongPeriod = time.Second * 30  // Shorter timeout

## Frontend Integration

The framework is designed to work seamlessly with JavaScript WebSocket clients:

	class WebSocketClient {
		constructor(url) {
			this.url = url;
			this.ws = null;
			this.pingInterval = null;
		}

		connect() {
			this.ws = new WebSocket(this.url);

			this.ws.onopen = () => {
				this.startPingInterval();
			};

			this.ws.onmessage = (event) => {
				const message = JSON.parse(event.data);
				this.handleMessage(message);
			};
		}

		handleMessage(message) {
			switch (message.type) {
				case 'ping':
					// Respond to server ping
					this.send({type: 'pong', pingId: message.pingId});
					break;
				default:
					this.onMessage(message);
			}
		}

		startPingInterval() {
			this.pingInterval = setInterval(() => {
				if (this.ws?.readyState === WebSocket.OPEN) {
					this.send({type: 'ping', timestamp: Date.now()});
				}
			}, 25000);
		}
	}

## Error Handling and Resilience

The framework provides robust error handling:

	type ResilientConn struct {
		gohttp.JSONConn
		errorCount int
		maxErrors  int
	}

	func (r *ResilientConn) OnError(err error) error {
		r.errorCount++
		log.Printf("WebSocket error #%d: %v", r.errorCount, err)

		if r.errorCount > r.maxErrors {
			return err // Close connection after too many errors
		}

		return nil // Continue with connection
	}

	func (r *ResilientConn) OnTimeout() bool {
		log.Printf("Connection timeout for %s", r.ConnId())
		return true // Close the connection
	}

## Best Practices

  - Always call parent OnStart/OnClose when embedding JSONConn
  - Implement proper authentication in the Validate method
  - Use connection limits to prevent resource exhaustion
  - Handle errors gracefully without crashing the server
  - Clean up resources properly in OnClose methods
  - Use mutexes for thread-safe operations on shared state
  - Monitor connection health with metrics

## Common Patterns

### Hub Pattern for Broadcasting

	type Hub struct {
		clients    map[*Client]bool
		broadcast  chan []byte
		register   chan *Client
		unregister chan *Client
	}

	func (h *Hub) run() {
		for {
			select {
			case client := <-h.register:
				h.clients[client] = true
			case client := <-h.unregister:
				delete(h.clients, client)
			case message := <-h.broadcast:
				for client := range h.clients {
					client.send <- message
				}
			}
		}
	}

### Room-based Messaging

	type Room struct {
		name    string
		clients map[string]*Client
		mu      sync.RWMutex
	}

	func (r *Room) Broadcast(message any, excludeId string) {
		r.mu.RLock()
		defer r.mu.RUnlock()

		for id, client := range r.clients {
			if id != excludeId {
				client.Send(message)
			}
		}
	}

## Testing

The package includes comprehensive test utilities for WebSocket testing:

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/endpoint"
	conn, err := createTestClient(t, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

See the comprehensive test suite in ws2_test.go for complete examples of testing
WebSocket applications including authentication, load testing, and error scenarios.

# HTTP Utilities

The package also provides utilities for HTTP request/response handling:

  - JsonToQueryString: Convert maps to URL query strings
  - SendJsonResponse: Send JSON responses with proper error handling
  - ErrorToHttpCode: Convert Go errors to appropriate HTTP status codes
  - WSConnWriteMessage/WSConnWriteError: WebSocket message writing utilities
  - NormalizeWsUrl: Convert HTTP URLs to WebSocket URLs

Example HTTP utility usage:

	func handleAPI(w http.ResponseWriter, r *http.Request) {
		data, err := processRequest(r)
		gohttp.SendJsonResponse(w, data, err)
	}

This comprehensive framework enables building production-ready real-time applications
with WebSocket communication, from simple echo servers to complex multi-user systems
with authentication, rooms, and advanced connection management.
*/
package http
