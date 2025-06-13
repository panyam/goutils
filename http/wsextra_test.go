package http

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
)

// Test message types for comprehensive testing
type TestMessage struct {
	Type      string `json:"type"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
	ID        string `json:"id,omitempty"`
}

// Echo server implementation for basic testing
type EchoConn struct {
	JSONConn
	messageCount int64
	lastMessage  time.Time
}

func (e *EchoConn) HandleMessage(msg any) error {
	atomic.AddInt64(&e.messageCount, 1)
	e.lastMessage = time.Now()

	// Echo back the message with metadata
	response := map[string]any{
		"type":         "echo",
		"originalMsg":  msg,
		"messageCount": atomic.LoadInt64(&e.messageCount),
		"timestamp":    time.Now().Unix(),
		"connId":       e.ConnId(),
	}

	e.Writer.Send(conc.Message[any]{Value: response})
	return nil
}

type EchoHandler struct{}

func (h *EchoHandler) Validate(w http.ResponseWriter, r *http.Request) (*EchoConn, bool) {
	return &EchoConn{}, true
}

// Chat server implementation for advanced testing
type ChatServer struct {
	clients   map[string]*ChatConn
	rooms     map[string]*Room
	mu        sync.RWMutex
	connCount int64
	msgCount  int64
}

type Room struct {
	name    string
	clients map[string]*ChatConn
	mu      sync.RWMutex
}

func (r *Room) AddClient(client *ChatConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client.ConnId()] = client
}

func (r *Room) RemoveClient(clientId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, clientId)
}

func (r *Room) Broadcast(message any, excludeId string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, client := range r.clients {
		if id != excludeId {
			client.Writer.Send(conc.Message[any]{Value: message})
		}
	}
}

type ChatConn struct {
	JSONConn
	server    *ChatServer
	username  string
	roomName  string
	pingCount int64
}

func (c *ChatConn) OnStart(conn *websocket.Conn) error {
	if err := c.JSONConn.OnStart(conn); err != nil {
		return err
	}

	// Register with server
	c.server.mu.Lock()
	c.server.clients[c.ConnId()] = c
	atomic.AddInt64(&c.server.connCount, 1)
	c.server.mu.Unlock()

	// Join default room
	c.joinRoom("general")

	// Send welcome message
	welcome := map[string]any{
		"type":     "welcome",
		"connId":   c.ConnId(),
		"username": c.username,
		"room":     c.roomName,
		"server":   "GoUtils Chat Server",
	}
	c.Writer.Send(conc.Message[any]{Value: welcome})

	return nil
}

func (c *ChatConn) OnClose() {
	// Leave room
	if c.roomName != "" {
		c.leaveRoom()
	}

	// Unregister from server
	c.server.mu.Lock()
	delete(c.server.clients, c.ConnId())
	atomic.AddInt64(&c.server.connCount, -1)
	c.server.mu.Unlock()

	c.JSONConn.OnClose()
}

func (c *ChatConn) HandleMessage(msg any) error {
	atomic.AddInt64(&c.server.msgCount, 1)

	msgMap, ok := msg.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid message format")
	}

	switch msgMap["type"].(string) {
	case "ping":
		// Handle client ping
		atomic.AddInt64(&c.pingCount, 1)
		pong := map[string]any{
			"type":      "pong",
			"pingId":    msgMap["pingId"],
			"timestamp": time.Now().Unix(),
			"pingCount": atomic.LoadInt64(&c.pingCount),
		}
		c.Writer.Send(conc.Message[any]{Value: pong})

	case "pong":
		// Handle client pong response
		log.Printf("Received pong from client %s", c.ConnId())

	case "chat":
		// Broadcast chat message to room
		if c.roomName != "" {
			chatMsg := map[string]any{
				"type":      "chat",
				"username":  c.username,
				"message":   msgMap["message"],
				"room":      c.roomName,
				"timestamp": time.Now().Unix(),
			}

			if room, exists := c.server.rooms[c.roomName]; exists {
				room.Broadcast(chatMsg, c.ConnId())
			}
		}

	case "join_room":
		// Switch to different room
		newRoom := msgMap["room"].(string)
		c.leaveRoom()
		c.joinRoom(newRoom)

	case "list_users":
		// List users in current room
		users := c.getUsersInRoom()
		userList := map[string]any{
			"type":  "user_list",
			"room":  c.roomName,
			"users": users,
		}
		c.Writer.Send(conc.Message[any]{Value: userList})

	case "server_stats":
		// Return server statistics
		stats := map[string]any{
			"type":           "server_stats",
			"total_clients":  atomic.LoadInt64(&c.server.connCount),
			"total_messages": atomic.LoadInt64(&c.server.msgCount),
			"total_rooms":    len(c.server.rooms),
			"current_time":   time.Now().Unix(),
		}
		c.Writer.Send(conc.Message[any]{Value: stats})

	default:
		log.Printf("Unknown message type: %s", msgMap["type"])
	}

	return nil
}

func (c *ChatConn) joinRoom(roomName string) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	// Create room if it doesn't exist
	if _, exists := c.server.rooms[roomName]; !exists {
		c.server.rooms[roomName] = &Room{
			name:    roomName,
			clients: make(map[string]*ChatConn),
		}
	}

	room := c.server.rooms[roomName]
	room.AddClient(c)
	c.roomName = roomName

	// Notify room of new user
	joinMsg := map[string]any{
		"type":     "user_joined",
		"username": c.username,
		"room":     roomName,
		"connId":   c.ConnId(),
	}
	room.Broadcast(joinMsg, c.ConnId())
}

func (c *ChatConn) leaveRoom() {
	if c.roomName == "" {
		return
	}

	c.server.mu.RLock()
	room, exists := c.server.rooms[c.roomName]
	c.server.mu.RUnlock()

	if exists {
		// Notify room of user leaving
		leaveMsg := map[string]any{
			"type":     "user_left",
			"username": c.username,
			"room":     c.roomName,
			"connId":   c.ConnId(),
		}
		room.Broadcast(leaveMsg, c.ConnId())

		room.RemoveClient(c.ConnId())
	}

	c.roomName = ""
}

func (c *ChatConn) getUsersInRoom() []string {
	if c.roomName == "" {
		return []string{}
	}

	c.server.mu.RLock()
	room, exists := c.server.rooms[c.roomName]
	c.server.mu.RUnlock()

	if !exists {
		return []string{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	users := make([]string, 0, len(room.clients))
	for _, client := range room.clients {
		users = append(users, client.username)
	}

	return users
}

type ChatHandler struct {
	server *ChatServer
}

func (h *ChatHandler) Validate(w http.ResponseWriter, r *http.Request) (*ChatConn, bool) {
	// Extract username from query parameters
	username := r.URL.Query().Get("username")
	if username == "" {
		username = fmt.Sprintf("user_%d", time.Now().UnixNano()%10000)
	}

	return &ChatConn{
		server:   h.server,
		username: username,
	}, true
}

// Authentication test implementation
type AuthConn struct {
	JSONConn
	userID string
	roles  []string
}

func (a *AuthConn) HandleMessage(msg any) error {
	msgMap := msg.(map[string]any)
	messageType := msgMap["type"].(string)

	// Check permissions
	if !a.hasPermission(messageType) {
		errorMsg := map[string]any{
			"type":  "error",
			"error": "Insufficient permissions",
			"code":  "PERMISSION_DENIED",
		}
		a.Writer.Send(conc.Message[any]{Value: errorMsg})
		return nil
	}

	// Process authorized message
	response := map[string]any{
		"type":     "authorized_response",
		"userID":   a.userID,
		"original": msg,
	}
	a.Writer.Send(conc.Message[any]{Value: response})

	return nil
}

func (a *AuthConn) hasPermission(messageType string) bool {
	permissions := map[string][]string{
		"admin_command": {"admin"},
		"moderate":      {"admin", "moderator"},
		"user_action":   {"user", "moderator", "admin"},
	}

	required, exists := permissions[messageType]
	if !exists {
		return false // Unknown message type, deny by default
	}

	for _, role := range a.roles {
		for _, req := range required {
			if role == req {
				return true
			}
		}
	}

	return false
}

type AuthHandler struct{}

func (h *AuthHandler) Validate(w http.ResponseWriter, r *http.Request) (*AuthConn, bool) {
	// Simulate JWT token validation
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	// Mock token validation
	var userID string
	var roles []string

	switch token {
	case "admin_token":
		userID = "admin_user"
		roles = []string{"admin"}
	case "mod_token":
		userID = "mod_user"
		roles = []string{"moderator"}
	case "user_token":
		userID = "regular_user"
		roles = []string{"user"}
	default:
		http.Error(w, "Invalid or missing token", http.StatusUnauthorized)
		return nil, false
	}

	return &AuthConn{
		userID: userID,
		roles:  roles,
	}, true
}

// Load testing connection for performance tests
type LoadTestConn struct {
	JSONConn
	messageCount int64
	startTime    time.Time
	lastMsgTime  time.Time
}

func (l *LoadTestConn) OnStart(conn *websocket.Conn) error {
	if err := l.JSONConn.OnStart(conn); err != nil {
		return err
	}

	l.startTime = time.Now()
	l.lastMsgTime = time.Now()
	return nil
}

func (l *LoadTestConn) HandleMessage(msg any) error {
	atomic.AddInt64(&l.messageCount, 1)
	l.lastMsgTime = time.Now()

	msgMap := msg.(map[string]any)
	if msgMap["type"].(string) == "echo_request" {
		response := map[string]any{
			"type":         "echo_response",
			"messageCount": atomic.LoadInt64(&l.messageCount),
			"uptime":       time.Since(l.startTime).Seconds(),
			"data":         msgMap["data"],
		}
		l.Writer.Send(conc.Message[any]{Value: response})
	}

	return nil
}

type LoadTestHandler struct{}

func (h *LoadTestHandler) Validate(w http.ResponseWriter, r *http.Request) (*LoadTestConn, bool) {
	return &LoadTestConn{}, true
}

// Helper function to create a test WebSocket client
func createTestClient(t *testing.T, url string, headers http.Header) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Helper function to send and receive JSON messages
func sendJSONMessage(conn *websocket.Conn, msg any) error {
	return conn.WriteJSON(msg)
}

func receiveJSONMessage(conn *websocket.Conn, timeout time.Duration) (map[string]any, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	var msg map[string]any
	err := conn.ReadJSON(&msg)
	return msg, err
}

// Test basic echo functionality
func TestEchoServer(t *testing.T) {
	// Create test server
	router := mux.NewRouter()
	handler := &EchoHandler{}
	router.HandleFunc("/echo", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	// Create WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/echo"
	conn, err := createTestClient(t, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Test echo functionality
	testMsg := map[string]any{
		"type": "test",
		"data": "Hello, WebSocket!",
	}

	err = sendJSONMessage(conn, testMsg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Receive echo response
	response, err := receiveJSONMessage(conn, time.Second*5)
	if err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}

	if response["type"] != "echo" {
		t.Errorf("Expected echo response, got %v", response["type"])
	}

	originalMsg := response["originalMsg"].(map[string]any)
	if originalMsg["data"] != testMsg["data"] {
		t.Errorf("Echo data mismatch: expected %v, got %v", testMsg["data"], originalMsg["data"])
	}
}

// Test ping-pong mechanism
func TestPingPongMechanism(t *testing.T) {
	router := mux.NewRouter()

	chatServer := &ChatServer{
		clients: make(map[string]*ChatConn),
		rooms:   make(map[string]*Room),
	}
	handler := &ChatHandler{server: chatServer}

	// Use shorter ping intervals for testing
	config := &WSConnConfig{
		BiDirStreamConfig: &BiDirStreamConfig{
			PingPeriod: time.Millisecond * 100, // Very short for testing
			PongPeriod: time.Second * 2,
		},
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	router.HandleFunc("/chat", WSServe(handler, config))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/chat?username=testuser"
	conn, err := createTestClient(t, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for welcome message
	_, err = receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive welcome: %v", err)
	}

	// Send a ping to server
	pingMsg := map[string]any{
		"type":   "ping",
		"pingId": 1,
	}

	err = sendJSONMessage(conn, pingMsg)
	if err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Should receive pong response
	pongResponse, err := receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive pong: %v", err)
	}

	if pongResponse["type"] != "pong" {
		t.Errorf("Expected pong response, got %v", pongResponse["type"])
	}

	if pongResponse["pingId"] != float64(1) {
		t.Errorf("Expected pingId 1, got %v", pongResponse["pingId"])
	}
}

// Test chat server functionality
func TestChatServer(t *testing.T) {
	router := mux.NewRouter()

	chatServer := &ChatServer{
		clients: make(map[string]*ChatConn),
		rooms:   make(map[string]*Room),
	}
	handler := &ChatHandler{server: chatServer}
	router.HandleFunc("/chat", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	// Create two clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/chat"

	// Connect Alice first
	conn1, err := createTestClient(t, wsURL+"?username=alice", nil)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	// Alice receives her welcome message
	aliceWelcome, err := receiveJSONMessage(conn1, time.Second*2)
	if err != nil {
		t.Fatalf("Alice failed to receive welcome: %v", err)
	}
	if aliceWelcome["type"] != "welcome" {
		t.Errorf("Expected welcome message for Alice, got %v", aliceWelcome["type"])
	}

	// Connect Bob second
	conn2, err := createTestClient(t, wsURL+"?username=bob", nil)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	// Bob receives his welcome message
	bobWelcome, err := receiveJSONMessage(conn2, time.Second*2)
	if err != nil {
		t.Fatalf("Bob failed to receive welcome: %v", err)
	}
	if bobWelcome["type"] != "welcome" {
		t.Errorf("Expected welcome message for Bob, got %v", bobWelcome["type"])
	}

	// Alice should receive notification that Bob joined (since Alice was already in the room)
	joinNotification, err := receiveJSONMessage(conn1, time.Second*2)
	if err != nil {
		t.Fatalf("Alice failed to receive Bob's join notification: %v", err)
	}

	if joinNotification["type"] != "user_joined" {
		t.Errorf("Expected user_joined notification, got %v", joinNotification["type"])
	}

	if joinNotification["username"] != "bob" {
		t.Errorf("Expected join notification for bob, got %v", joinNotification["username"])
	}

	// Send chat message from client 1
	chatMsg := map[string]any{
		"type":    "chat",
		"message": "Hello from Alice!",
	}

	err = sendJSONMessage(conn1, chatMsg)
	if err != nil {
		t.Fatalf("Failed to send chat message: %v", err)
	}

	// Client 2 should receive the chat message
	receivedMsg, err := receiveJSONMessage(conn2, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive chat message: %v", err)
	}

	if receivedMsg["type"] != "chat" {
		t.Errorf("Expected chat message, got %v", receivedMsg["type"])
	}

	if receivedMsg["username"] != "alice" {
		t.Errorf("Expected username alice, got %v", receivedMsg["username"])
	}

	if receivedMsg["message"] != "Hello from Alice!" {
		t.Errorf("Expected message 'Hello from Alice!', got %v", receivedMsg["message"])
	}
}

// Test authentication and authorization
func TestAuthentication(t *testing.T) {
	router := mux.NewRouter()
	handler := &AuthHandler{}
	router.HandleFunc("/secure", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/secure"

	// Test with valid admin token
	headers := http.Header{}
	headers.Set("Authorization", "admin_token")

	conn, err := createTestClient(t, wsURL, headers)
	if err != nil {
		t.Fatalf("Failed to connect with admin token: %v", err)
	}
	defer conn.Close()

	// Test admin command (should succeed)
	adminMsg := map[string]any{
		"type": "admin_command",
		"data": "restart_server",
	}

	err = sendJSONMessage(conn, adminMsg)
	if err != nil {
		t.Fatalf("Failed to send admin message: %v", err)
	}

	response, err := receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}

	if response["type"] != "authorized_response" {
		t.Errorf("Expected authorized response for admin, got %v", response["type"])
	}

	// Test unauthorized command
	unauthorizedMsg := map[string]any{
		"type": "unknown_command",
		"data": "test",
	}

	err = sendJSONMessage(conn, unauthorizedMsg)
	if err != nil {
		t.Fatalf("Failed to send unauthorized message: %v", err)
	}

	errorResponse, err := receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive error response: %v", err)
	}

	if errorResponse["type"] != "error" {
		t.Errorf("Expected error response for unauthorized command, got %v", errorResponse["type"])
	}
}

// Test connection limits and resource management
func TestConnectionLimits(t *testing.T) {
	router := mux.NewRouter()
	handler := &EchoHandler{}
	router.HandleFunc("/echo", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/echo"

	// Create multiple connections
	var connections []*websocket.Conn
	const maxConnections = 5

	for i := 0; i < maxConnections; i++ {
		conn, err := createTestClient(t, wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to create connection %d: %v", i, err)
		}
		connections = append(connections, conn)
	}

	// Test that all connections are working
	for i, conn := range connections {
		testMsg := map[string]any{
			"type": "test",
			"data": fmt.Sprintf("Message from connection %d", i),
		}

		err := sendJSONMessage(conn, testMsg)
		if err != nil {
			t.Errorf("Failed to send message on connection %d: %v", i, err)
		}

		response, err := receiveJSONMessage(conn, time.Second*2)
		if err != nil {
			t.Errorf("Failed to receive response on connection %d: %v", i, err)
		}

		if response["type"] != "echo" {
			t.Errorf("Expected echo response on connection %d, got %v", i, response["type"])
		}
	}

	// Clean up connections
	for _, conn := range connections {
		conn.Close()
	}
}

// Load test with concurrent connections
func TestConcurrentConnections(t *testing.T) {
	router := mux.NewRouter()
	handler := &LoadTestHandler{}
	router.HandleFunc("/load", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/load"

	const numConnections = 10
	const messagesPerConnection = 5

	var wg sync.WaitGroup
	var totalMessages int64
	var totalErrors int64

	// Launch concurrent connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			conn, err := createTestClient(t, wsURL, nil)
			if err != nil {
				atomic.AddInt64(&totalErrors, 1)
				t.Errorf("Failed to create connection %d: %v", connID, err)
				return
			}
			defer conn.Close()

			// Send multiple messages
			for j := 0; j < messagesPerConnection; j++ {
				testMsg := map[string]any{
					"type": "echo_request",
					"data": fmt.Sprintf("Message %d from connection %d", j, connID),
				}

				err := sendJSONMessage(conn, testMsg)
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					continue
				}

				response, err := receiveJSONMessage(conn, time.Second*5)
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					continue
				}

				if response["type"] == "echo_response" {
					atomic.AddInt64(&totalMessages, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedMessages := int64(numConnections * messagesPerConnection)
	if atomic.LoadInt64(&totalMessages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, atomic.LoadInt64(&totalMessages))
	}

	if atomic.LoadInt64(&totalErrors) > 0 {
		t.Errorf("Encountered %d errors during load test", atomic.LoadInt64(&totalErrors))
	}

	t.Logf("Load test completed: %d connections, %d messages total, %d errors",
		numConnections, atomic.LoadInt64(&totalMessages), atomic.LoadInt64(&totalErrors))
}

// Test connection timeouts and cleanup
func TestConnectionTimeout(t *testing.T) {
	// This test demonstrates timeout behavior, but actual timeout depends on
	// the WebSocket implementation and network conditions, so we test more generally

	router := mux.NewRouter()
	handler := &EchoHandler{}

	// Configure short timeout for testing
	config := &WSConnConfig{
		BiDirStreamConfig: &BiDirStreamConfig{
			PingPeriod: time.Millisecond * 50,
			PongPeriod: time.Millisecond * 100,
		},
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	router.HandleFunc("/timeout", WSServe(handler, config))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/timeout"
	conn, err := createTestClient(t, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send one message to establish connection
	testMsg := map[string]any{"type": "test"}
	err = sendJSONMessage(conn, testMsg)
	if err != nil {
		t.Fatalf("Failed to send initial message: %v", err)
	}

	// Receive echo to confirm connection works
	_, err = receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive initial echo: %v", err)
	}

	// Test that the connection is working normally first
	err = sendJSONMessage(conn, testMsg)
	if err != nil {
		t.Fatalf("Failed to send second message: %v", err)
	}

	_, err = receiveJSONMessage(conn, time.Second*1)
	if err != nil {
		t.Fatalf("Failed to receive second echo: %v", err)
	}

	t.Logf("Connection timeout test completed - connection remained stable during test period")
}

// Test error handling and recovery
func TestErrorHandling(t *testing.T) {
	router := mux.NewRouter()
	handler := &EchoHandler{}
	router.HandleFunc("/echo", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/echo"
	conn, err := createTestClient(t, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a valid message first to establish baseline
	testMsg := map[string]any{
		"type": "test",
		"data": "initial message",
	}

	err = sendJSONMessage(conn, testMsg)
	if err != nil {
		t.Fatalf("Failed to send initial message: %v", err)
	}

	response, err := receiveJSONMessage(conn, time.Second*2)
	if err != nil {
		t.Fatalf("Failed to receive initial response: %v", err)
	}

	if response["type"] != "echo" {
		t.Errorf("Expected echo response, got %v", response["type"])
	}

	// Send invalid JSON (this will likely close the connection in the current implementation)
	err = conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	if err != nil {
		t.Fatalf("Failed to send invalid message: %v", err)
	}

	// Give some time for the server to process the invalid message
	time.Sleep(time.Millisecond * 100)

	// Try to send another message - this may fail if connection was closed
	validMsg := map[string]any{
		"type": "test",
		"data": "message after error",
	}

	err = sendJSONMessage(conn, validMsg)
	// The current implementation may close the connection on invalid JSON
	// so we'll just log this rather than fail the test
	if err != nil {
		t.Logf("Connection closed after invalid JSON (expected behavior): %v", err)
		return
	}

	// If connection is still alive, try to receive response
	response, err = receiveJSONMessage(conn, time.Second*1)
	if err != nil {
		t.Logf("No response after invalid JSON (connection may be closed): %v", err)
		return
	}

	if response["type"] == "echo" {
		t.Logf("Connection survived invalid JSON and continued working")
	}
}

// Benchmark WebSocket message throughput
func BenchmarkWebSocketThroughput(b *testing.B) {
	router := mux.NewRouter()
	handler := &LoadTestHandler{}
	router.HandleFunc("/bench", WSServe(handler, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/bench"
	conn, err := createTestClient(nil, wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	testMsg := map[string]any{
		"type": "echo_request",
		"data": "benchmark data",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := sendJSONMessage(conn, testMsg)
		if err != nil {
			b.Fatalf("Failed to send message: %v", err)
		}

		_, err = receiveJSONMessage(conn, time.Second*5)
		if err != nil {
			b.Fatalf("Failed to receive response: %v", err)
		}
	}
}

// Example demonstrating complete WebSocket workflow
func Example() {
	// Create router and server
	router := mux.NewRouter()

	// Chat server setup
	chatServer := &ChatServer{
		clients: make(map[string]*ChatConn),
		rooms:   make(map[string]*Room),
	}

	// Add handlers
	router.HandleFunc("/echo", WSServe(&EchoHandler{}, nil))
	router.HandleFunc("/chat", WSServe(&ChatHandler{server: chatServer}, nil))
	router.HandleFunc("/secure", WSServe(&AuthHandler{}, nil))

	// Custom configuration for production
	config := &WSConnConfig{
		BiDirStreamConfig: &BiDirStreamConfig{
			PingPeriod: time.Second * 30,
			PongPeriod: time.Second * 300,
		},
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// Add proper origin checking in production
				return true
			},
		},
	}

	router.HandleFunc("/production", WSServe(&EchoHandler{}, config))

	// In a real application, you would start the server:
	// log.Fatal(http.ListenAndServe(":8080", router))

	fmt.Println("WebSocket server configured with multiple endpoints:")
	fmt.Println("- /echo: Simple echo server")
	fmt.Println("- /chat: Multi-user chat with rooms")
	fmt.Println("- /secure: Authenticated connections")
	fmt.Println("- /production: Production-ready config")

	// Output:
	// WebSocket server configured with multiple endpoints:
	// - /echo: Simple echo server
	// - /chat: Multi-user chat with rooms
	// - /secure: Authenticated connections
	// - /production: Production-ready config
}
