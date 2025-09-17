package domain

import (
	"time"
)

// Challenge represents a captcha challenge
type Challenge struct {
	ID         string            `json:"id"`
	Type       ChallengeType     `json:"type"`
	Complexity int32             `json:"complexity"`
	HTML       string            `json:"html"`
	Answer     interface{}       `json:"-"` // Hidden from JSON
	CreatedAt  time.Time         `json:"created_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Solved     bool              `json:"solved"`
	Metadata   map[string]string `json:"metadata"`
}

// ChallengeType represents the type of captcha challenge
type ChallengeType string

const (
	ChallengeTypeDragDrop ChallengeType = "drag_drop"
	ChallengeTypeClick    ChallengeType = "click"
	ChallengeTypeSwipe    ChallengeType = "swipe"
	ChallengeTypeGame     ChallengeType = "game"
)

// ChallengeResult represents the result of solving a challenge
type ChallengeResult struct {
	ChallengeID       string `json:"challenge_id"`
	Solved            bool   `json:"solved"`
	ConfidencePercent int32  `json:"confidence_percent"`
	TimeToSolve       int64  `json:"time_to_solve_ms"`
	Attempts          int32  `json:"attempts"`
	Error             string `json:"error,omitempty"`
}

// Event represents a client or server event
type Event struct {
	Type        EventType `json:"type"`
	ChallengeID string    `json:"challenge_id"`
	Data        []byte    `json:"data"`
	Timestamp   time.Time `json:"timestamp"`
}

// EventType represents the type of event
type EventType string

const (
	EventTypeFrontendEvent    EventType = "frontend_event"
	EventTypeConnectionClosed EventType = "connection_closed"
	EventTypeBalancerEvent    EventType = "balancer_event"
)

// ServerEvent represents an event sent from server to client
type ServerEvent struct {
	Type              ServerEventType `json:"type"`
	ChallengeID       string          `json:"challenge_id"`
	Data              []byte          `json:"data"`
	JSCode            string          `json:"js_code,omitempty"`
	ConfidencePercent int32           `json:"confidence_percent,omitempty"`
	Timestamp         time.Time       `json:"timestamp"`
}

// ServerEventType represents the type of server event
type ServerEventType string

const (
	ServerEventTypeChallengeResult ServerEventType = "challenge_result"
	ServerEventTypeRunClientJS     ServerEventType = "run_client_js"
	ServerEventTypeSendClientData  ServerEventType = "send_client_data"
)
