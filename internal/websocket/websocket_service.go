package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WebSocketService handles WebSocket connections and events
type WebSocketService struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	eventBus    chan *Event
	handlers    map[string]EventHandler
}

// Connection represents a WebSocket connection
type Connection struct {
	ID        string
	ClientID  string
	CreatedAt time.Time
	LastSeen  time.Time
	Active    bool
	Events    chan *Event
}

// Event represents a WebSocket event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	ClientID  string                 `json:"client_id,omitempty"`
}

// EventHandler handles specific event types
type EventHandler func(ctx context.Context, event *Event) error

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService() *WebSocketService {
	ws := &WebSocketService{
		connections: make(map[string]*Connection),
		eventBus:    make(chan *Event, 1000),
		handlers:    make(map[string]EventHandler),
	}
	
	// Start event processing
	go ws.processEvents()
	
	return ws
}

// RegisterHandler registers an event handler
func (ws *WebSocketService) RegisterHandler(eventType string, handler EventHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	ws.handlers[eventType] = handler
}

// CreateConnection creates a new WebSocket connection
func (ws *WebSocketService) CreateConnection(clientID string) *Connection {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	connID := uuid.New().String()
	conn := &Connection{
		ID:        connID,
		ClientID:  clientID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Active:    true,
		Events:    make(chan *Event, 100),
	}
	
	ws.connections[connID] = conn
	
	return conn
}

// GetConnection retrieves a connection by ID
func (ws *WebSocketService) GetConnection(connID string) (*Connection, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	
	conn, exists := ws.connections[connID]
	return conn, exists
}

// SendEvent sends an event to a specific connection
func (ws *WebSocketService) SendEvent(connID string, event *Event) error {
	ws.mu.RLock()
	conn, exists := ws.connections[connID]
	ws.mu.RUnlock()
	
	if !exists || !conn.Active {
		return fmt.Errorf("connection not found or inactive")
	}
	
	select {
	case conn.Events <- event:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending event")
	}
}

// BroadcastEvent broadcasts an event to all active connections
func (ws *WebSocketService) BroadcastEvent(event *Event) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	
	for _, conn := range ws.connections {
		if conn.Active {
			select {
			case conn.Events <- event:
			default:
				// Skip if channel is full
			}
		}
	}
}

// BroadcastToClient broadcasts an event to all connections of a specific client
func (ws *WebSocketService) BroadcastToClient(clientID string, event *Event) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	
	for _, conn := range ws.connections {
		if conn.Active && conn.ClientID == clientID {
			select {
			case conn.Events <- event:
			default:
				// Skip if channel is full
			}
		}
	}
}

// CloseConnection closes a WebSocket connection
func (ws *WebSocketService) CloseConnection(connID string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	if conn, exists := ws.connections[connID]; exists {
		conn.Active = false
		close(conn.Events)
		delete(ws.connections, connID)
	}
}

// ProcessMessage processes a message from a client
func (ws *WebSocketService) ProcessMessage(ctx context.Context, connID string, message []byte) error {
	var event Event
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	// Set connection ID
	event.ClientID = connID
	
	// Add to event bus
	select {
	case ws.eventBus <- &event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("event bus is full")
	}
}

// processEvents processes events from the event bus
func (ws *WebSocketService) processEvents() {
	for event := range ws.eventBus {
		ws.handleEvent(context.Background(), event)
	}
}

// handleEvent handles a specific event
func (ws *WebSocketService) handleEvent(ctx context.Context, event *Event) {
	ws.mu.RLock()
	handler, exists := ws.handlers[event.Type]
	ws.mu.RUnlock()
	
	if !exists {
		// No handler for this event type
		return
	}
	
	// Execute handler
	if err := handler(ctx, event); err != nil {
		// Log error or handle it appropriately
		fmt.Printf("Error handling event %s: %v\n", event.Type, err)
	}
}

// GetConnectionStats returns statistics about connections
func (ws *WebSocketService) GetConnectionStats() map[string]interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	
	activeCount := 0
	totalCount := len(ws.connections)
	
	for _, conn := range ws.connections {
		if conn.Active {
			activeCount++
		}
	}
	
	return map[string]interface{}{
		"total_connections":  totalCount,
		"active_connections": activeCount,
		"event_bus_size":    len(ws.eventBus),
	}
}

// CleanupInactiveConnections removes inactive connections
func (ws *WebSocketService) CleanupInactiveConnections() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	now := time.Now()
	for connID, conn := range ws.connections {
		// Remove connections inactive for more than 1 hour
		if !conn.Active || now.Sub(conn.LastSeen) > time.Hour {
			close(conn.Events)
			delete(ws.connections, connID)
		}
	}
}

// StartCleanupRoutine starts a background cleanup routine
func (ws *WebSocketService) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.CleanupInactiveConnections()
		}
	}
}
