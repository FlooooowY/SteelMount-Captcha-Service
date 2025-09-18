package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// HTTPServer handles HTTP to WebSocket upgrades
type HTTPServer struct {
	wsService   *WebSocketService
	upgrader    websocket.Upgrader
	port        int
	server      *http.Server
}

// NewHTTPServer creates a new HTTP server for WebSocket connections
func NewHTTPServer(wsService *WebSocketService, port int) *HTTPServer {
	return &HTTPServer{
		wsService: wsService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		port: port,
	}
}

// GetWebSocketService returns the WebSocket service
func (s *HTTPServer) GetWebSocketService() *WebSocketService {
	return s.wsService
}

// Start starts the HTTP server
func (s *HTTPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)
	
	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)
	
	// Stats endpoint
	mux.HandleFunc("/stats", s.handleStats)
	
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	
	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()
	
	log.Printf("WebSocket server started on port %d", s.port)
	return nil
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleWebSocket handles WebSocket connections
func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract client ID from query parameters
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id parameter required", http.StatusBadRequest)
		return
	}
	
	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	
	// Create connection in service
	wsConn := s.wsService.CreateConnection(clientID)
	
	// Handle connection
	s.handleConnection(wsConn, conn)
}

// handleConnection handles a WebSocket connection
func (s *HTTPServer) handleConnection(wsConn *Connection, conn *websocket.Conn) {
	// Send connection established event
	event := &Event{
		ID:        fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		Type:      "connection_established",
		Data:      map[string]interface{}{"connection_id": wsConn.ID},
		Timestamp: time.Now(),
		ClientID:  wsConn.ClientID,
	}
	
	if err := s.sendEvent(conn, event); err != nil {
		log.Printf("Error sending connection event: %v", err)
		return
	}
	
	// Start goroutines for reading and writing
	done := make(chan struct{})
	
	// Read messages from client
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}
			
			// Process message
			if err := s.wsService.ProcessMessage(context.Background(), wsConn.ID, message); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()
	
	// Write events to client
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case event := <-wsConn.Events:
				if err := s.sendEvent(conn, event); err != nil {
					log.Printf("Error sending event: %v", err)
					return
				}
			case <-ticker.C:
				// Send ping
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	
	// Wait for connection to close
	<-done
	
	// Close connection in service
	s.wsService.CloseConnection(wsConn.ID)
}

// sendEvent sends an event to the WebSocket connection
func (s *HTTPServer) sendEvent(conn *websocket.Conn, event *Event) error {
	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Failed to set write deadline: %v", err)
	}
	
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	// Send as text message
	return conn.WriteMessage(websocket.TextMessage, data)
}

// handleHealth handles health check requests
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Unix(),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// handleStats handles stats requests
func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	stats := s.wsService.GetConnectionStats()
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode stats: %v", err)
	}
}
