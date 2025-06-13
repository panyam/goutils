# GoUtils WebSocket Tutorial

A comprehensive guide to using the goutils WebSocket package for production-grade real-time applications.

## Overview

The goutils WebSocket package provides a robust, production-ready framework for handling WebSocket connections with automatic connection management, ping-pong heartbeats, and lifecycle hooks. It's built on top of the Gorilla WebSocket library with additional abstractions for concurrent message handling.

## Key Features

- **Production-grade connection management** with automatic heartbeat detection
- **Lifecycle hooks** for connection start, close, timeout, and error handling
- **Thread-safe message broadcasting** to multiple clients
- **Automatic ping-pong mechanism** to prevent connection timeouts
- **Type-safe message handling** with generics
- **Built-in JSON message support** with the JSONConn implementation
- **Configurable timeouts and intervals** for different deployment scenarios

## Architecture

### Core Interfaces

#### WSConn[I any]
The main connection interface that applications must implement:

```go
type WSConn[I any] interface {
    BiDirStreamConn[I]
    ReadMessage(w *websocket.Conn) (I, error)
    OnStart(conn *websocket.Conn) error
}
```

#### WSHandler[I any, S WSConn[I]]
Validates HTTP requests and creates WebSocket connections:

```go
type WSHandler[I any, S WSConn[I]] interface {
    Validate(w http.ResponseWriter, r *http.Request) (S, bool)
}
```

#### BiDirStreamConn[I any]
Provides lifecycle and message handling methods:

```go
type BiDirStreamConn[I any] interface {
    SendPing() error
    Name() string
    ConnId() string
    HandleMessage(msg I) error
    OnError(err error) error
    OnClose()
    OnTimeout() bool
}
```

## Basic Usage

### 1. Simple Echo Server

```go
package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux"
    gohttp "github.com/panyam/goutils/http"
)

// Use the built-in JSONConn for simple JSON message handling
type EchoConn struct {
    gohttp.JSONConn
}

func (e *EchoConn) HandleMessage(msg any) error {
    log.Printf("Received: %v", msg)
    // Echo the message back
    e.Writer.Send(conc.Message[any]{Value: msg})
    return nil
}

type EchoHandler struct{}

func (h *EchoHandler) Validate(w http.ResponseWriter, r *http.Request) (*EchoConn, bool) {
    // Accept all connections - add authentication here in production
    return &EchoConn{}, true
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/echo", gohttp.WSServe(&EchoHandler{}, nil))
    
    log.Println("Echo server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

### 2. Custom Message Types

```go
// Define your message structure
type GameMessage struct {
    Type      string      `json:"type"`
    PlayerID  string      `json:"playerId"`
    Data      any `json:"data,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

type GameConn struct {
    gohttp.JSONConn
    playerID string
    gameRoom string
}

func (g *GameConn) HandleMessage(msg any) error {
    // Parse the generic message into our GameMessage struct
    msgMap, ok := msg.(map[string]any)
    if !ok {
        return fmt.Errorf("invalid message format")
    }
    
    gameMsg := GameMessage{
        Type:      msgMap["type"].(string),
        PlayerID:  g.playerID,
        Timestamp: time.Now().Unix(),
    }
    
    switch gameMsg.Type {
    case "move":
        return g.handleMove(msgMap["data"])
    case "chat":
        return g.handleChat(msgMap["data"])
    default:
        log.Printf("Unknown message type: %s", gameMsg.Type)
    }
    
    return nil
}

func (g *GameConn) OnStart(conn *websocket.Conn) error {
    // Call parent OnStart first
    if err := g.JSONConn.OnStart(conn); err != nil {
        return err
    }
    
    // Send welcome message
    welcome := GameMessage{
        Type: "welcome",
        Data: map[string]any{
            "playerId": g.playerID,
            "gameRoom": g.gameRoom,
        },
        Timestamp: time.Now().Unix(),
    }
    
    g.Writer.Send(conc.Message[any]{Value: welcome})
    return nil
}
```

## Advanced Features

### 1. Connection Management and Broadcasting

```go
type ChatServer struct {
    clients map[string]*ChatConn
    mu      sync.RWMutex
}

type ChatConn struct {
    gohttp.JSONConn
    server   *ChatServer
    username string
    id       string
}

func (c *ChatConn) OnStart(conn *websocket.Conn) error {
    if err := c.JSONConn.OnStart(conn); err != nil {
        return err
    }
    
    c.id = fmt.Sprintf("user_%s", conn.RemoteAddr().String())
    
    // Register this connection
    c.server.mu.Lock()
    c.server.clients[c.id] = c
    c.server.mu.Unlock()
    
    // Notify others of new user
    c.server.broadcast("user_joined", map[string]any{
        "username": c.username,
        "userId":   c.id,
    }, c.id) // Exclude self
    
    return nil
}

func (c *ChatConn) OnClose() {
    // Unregister this connection
    c.server.mu.Lock()
    delete(c.server.clients, c.id)
    c.server.mu.Unlock()
    
    // Notify others of user leaving
    c.server.broadcast("user_left", map[string]any{
        "username": c.username,
        "userId":   c.id,
    }, c.id)
    
    c.JSONConn.OnClose()
}

func (c *ChatConn) HandleMessage(msg any) error {
    msgMap := msg.(map[string]any)
    
    switch msgMap["type"].(string) {
    case "chat_message":
        c.server.broadcast("chat_message", map[string]any{
            "username": c.username,
            "message":  msgMap["message"],
            "timestamp": time.Now().Unix(),
        }, "")
    }
    
    return nil
}

func (s *ChatServer) broadcast(messageType string, data any, excludeId string) {
    message := map[string]any{
        "type": messageType,
        "data": data,
    }
    
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    for id, client := range s.clients {
        if id != excludeId {
            client.Writer.Send(conc.Message[any]{Value: message})
        }
    }
}
```

### 2. Custom Ping-Pong Handling

```go
type CustomConn struct {
    gohttp.JSONConn
    lastPingTime time.Time
    pingCount    int64
}

func (c *CustomConn) SendPing() error {
    c.pingCount++
    c.lastPingTime = time.Now()
    
    pingMsg := map[string]any{
        "type":      "ping",
        "pingId":    c.pingCount,
        "timestamp": c.lastPingTime.Unix(),
        "connId":    c.ConnId(),
    }
    
    c.Writer.Send(conc.Message[any]{Value: pingMsg})
    return nil
}

func (c *CustomConn) HandleMessage(msg any) error {
    msgMap, ok := msg.(map[string]any)
    if !ok {
        return fmt.Errorf("invalid message format")
    }
    
    switch msgMap["type"].(string) {
    case "pong":
        // Handle pong response
        pingId := int64(msgMap["pingId"].(float64))
        latency := time.Now().Sub(c.lastPingTime)
        log.Printf("Received pong for ping %d, latency: %v", pingId, latency)
        return nil
    case "ping":
        // Respond to client ping
        pongMsg := map[string]any{
            "type":      "pong",
            "pingId":    msgMap["pingId"],
            "timestamp": time.Now().Unix(),
        }
        c.Writer.Send(conc.Message[any]{Value: pongMsg})
        return nil
    default:
        // Handle other message types
        return c.handleBusinessLogic(msgMap)
    }
}

func (c *CustomConn) OnTimeout() bool {
    log.Printf("Connection %s timed out after %d pings", c.ConnId(), c.pingCount)
    return true // Close the connection
}
```

### 3. Authentication and Authorization

```go
type AuthenticatedHandler struct {
    jwtSecret []byte
}

func (h *AuthenticatedHandler) Validate(w http.ResponseWriter, r *http.Request) (*SecureConn, bool) {
    // Extract token from header or query parameter
    token := r.Header.Get("Authorization")
    if token == "" {
        token = r.URL.Query().Get("token")
    }
    
    if token == "" {
        http.Error(w, "Missing authentication token", http.StatusUnauthorized)
        return nil, false
    }
    
    // Validate JWT token
    claims, err := h.validateJWT(token)
    if err != nil {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return nil, false
    }
    
    // Create authenticated connection
    conn := &SecureConn{
        userID:   claims["userId"].(string),
        username: claims["username"].(string),
        roles:    claims["roles"].([]string),
    }
    
    return conn, true
}

type SecureConn struct {
    gohttp.JSONConn
    userID   string
    username string
    roles    []string
}

func (s *SecureConn) HandleMessage(msg any) error {
    msgMap := msg.(map[string]any)
    
    // Check permissions for the message type
    if !s.hasPermission(msgMap["type"].(string)) {
        return fmt.Errorf("insufficient permissions")
    }
    
    // Process authorized message
    return s.processMessage(msgMap)
}

func (s *SecureConn) hasPermission(messageType string) bool {
    // Implement role-based access control
    requiredRoles := map[string][]string{
        "admin_command": {"admin"},
        "moderate":      {"admin", "moderator"},
        "chat":          {"user", "moderator", "admin"},
    }
    
    required, exists := requiredRoles[messageType]
    if !exists {
        return false
    }
    
    for _, role := range s.roles {
        for _, req := range required {
            if role == req {
                return true
            }
        }
    }
    
    return false
}
```

## Configuration

### Custom Configuration

```go
config := &gohttp.WSConnConfig{
    BiDirStreamConfig: &gohttp.BiDirStreamConfig{
        PingPeriod: time.Second * 25,  // Send ping every 25 seconds
        PongPeriod: time.Second * 300, // Timeout after 5 minutes of no activity
    },
    Upgrader: websocket.Upgrader{
        ReadBufferSize:  4096,
        WriteBufferSize: 4096,
        CheckOrigin: func(r *http.Request) bool {
            // Allow connections from specific origins
            origin := r.Header.Get("Origin")
            return origin == "https://yourdomain.com"
        },
    },
}

r.HandleFunc("/ws", gohttp.WSServe(handler, config))
```

### Environment-specific Configurations

```go
func getWSConfig() *gohttp.WSConnConfig {
    config := gohttp.DefaultWSConnConfig()
    
    if os.Getenv("ENV") == "production" {
        // More conservative timeouts for production
        config.PingPeriod = time.Second * 30
        config.PongPeriod = time.Second * 60
        config.Upgrader.CheckOrigin = func(r *http.Request) bool {
            // Strict origin checking in production
            origin := r.Header.Get("Origin")
            allowedOrigins := []string{
                "https://yourdomain.com",
                "https://app.yourdomain.com",
            }
            for _, allowed := range allowedOrigins {
                if origin == allowed {
                    return true
                }
            }
            return false
        }
    } else {
        // More lenient settings for development
        config.PingPeriod = time.Second * 10
        config.PongPeriod = time.Second * 30
    }
    
    return config
}
```

## Frontend Integration

### JavaScript Client with Ping-Pong

```javascript
class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.pingInterval = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
    }
    
    connect() {
        this.ws = new WebSocket(this.url);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.startPingInterval();
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.stopPingInterval();
            this.attemptReconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }
    
    handleMessage(message) {
        switch (message.type) {
            case 'ping':
                // Respond to server ping
                this.send({
                    type: 'pong',
                    pingId: message.pingId,
                    timestamp: Date.now()
                });
                break;
            case 'pong':
                // Handle server pong response
                console.log('Received pong from server');
                break;
            default:
                // Handle application-specific messages
                this.onMessage(message);
        }
    }
    
    startPingInterval() {
        // Send ping every 25 seconds to keep connection alive
        this.pingInterval = setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.send({
                    type: 'ping',
                    timestamp: Date.now()
                });
            }
        }, 25000);
    }
    
    stopPingInterval() {
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
        }
    }
    
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }
    
    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting reconnect ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
            setTimeout(() => this.connect(), 3000 * this.reconnectAttempts);
        }
    }
    
    // Override this method to handle application messages
    onMessage(message) {
        console.log('Received message:', message);
    }
    
    disconnect() {
        this.stopPingInterval();
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

// Usage
const client = new WebSocketClient('ws://localhost:8080/ws');
client.onMessage = (message) => {
    // Handle your application-specific messages
    console.log('Application message:', message);
};
client.connect();
```

## Error Handling and Resilience

### Graceful Error Handling

```go
type ResilientConn struct {
    gohttp.JSONConn
    errorCount    int
    maxErrors     int
    lastErrorTime time.Time
}

func (r *ResilientConn) OnError(err error) error {
    r.errorCount++
    r.lastErrorTime = time.Now()
    
    log.Printf("WebSocket error #%d: %v", r.errorCount, err)
    
    // Close connection after too many errors
    if r.errorCount > r.maxErrors {
        log.Printf("Too many errors (%d), closing connection", r.errorCount)
        return err // This will close the connection
    }
    
    // Reset error count after successful period
    if time.Since(r.lastErrorTime) > time.Minute*5 {
        r.errorCount = 0
    }
    
    return nil // Continue with connection
}

func (r *ResilientConn) OnTimeout() bool {
    log.Printf("Connection timeout for %s", r.ConnId())
    
    // Try to send a final message before closing
    r.Writer.Send(conc.Message[any]{Value: map[string]any{
        "type": "timeout_warning",
        "message": "Connection will be closed due to inactivity",
    }})
    
    return true // Close the connection
}
```

## Performance Optimization

### Connection Pooling and Resource Management

```go
type ConnectionManager struct {
    connections map[string]*ManagedConn
    mu          sync.RWMutex
    maxConns    int
    connCount   int64
}

func (cm *ConnectionManager) AddConnection(conn *ManagedConn) bool {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if len(cm.connections) >= cm.maxConns {
        log.Printf("Connection limit reached (%d), rejecting new connection", cm.maxConns)
        return false
    }
    
    cm.connections[conn.ConnId()] = conn
    atomic.AddInt64(&cm.connCount, 1)
    return true
}

func (cm *ConnectionManager) RemoveConnection(connId string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if _, exists := cm.connections[connId]; exists {
        delete(cm.connections, connId)
        atomic.AddInt64(&cm.connCount, -1)
    }
}

func (cm *ConnectionManager) BroadcastToRoom(roomId string, message any) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    for _, conn := range cm.connections {
        if conn.roomId == roomId {
            conn.Writer.Send(conc.Message[any]{Value: message})
        }
    }
}

func (cm *ConnectionManager) GetStats() map[string]any {
    return map[string]any{
        "total_connections": atomic.LoadInt64(&cm.connCount),
        "max_connections":   cm.maxConns,
    }
}
```

## Testing

See the comprehensive test file `ws2_test.go` for examples of:
- Unit testing WebSocket handlers
- Integration testing with real WebSocket connections
- Load testing with multiple concurrent connections
- Ping-pong mechanism testing
- Error scenario testing

## Best Practices

1. **Always call parent OnStart/OnClose**: When embedding JSONConn, call the parent methods first
2. **Handle ping-pong appropriately**: Implement custom ping-pong logic if needed for your use case
3. **Implement proper authentication**: Never trust client-side data, validate everything server-side
4. **Use connection limits**: Prevent resource exhaustion with maximum connection limits
5. **Graceful error handling**: Don't crash on client errors, log and continue
6. **Resource cleanup**: Always clean up resources in OnClose methods
7. **Thread safety**: Use mutexes when managing shared state across connections
8. **Monitor connection health**: Track metrics like connection count, error rates, and latency

## Common Patterns

### Hub Pattern for Broadcasting
Use a central hub to manage connections and broadcasting:

```go
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
```

### Room-based Messaging
Organize connections into rooms for targeted messaging:

```go
type Room struct {
    name    string
    clients map[string]*Client
    mu      sync.RWMutex
}

func (r *Room) AddClient(client *Client) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.clients[client.id] = client
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
```

This tutorial provides a solid foundation for building production-ready WebSocket applications with the goutils package. The key is to leverage the built-in lifecycle hooks and connection management while implementing your application-specific logic in the message handlers.
