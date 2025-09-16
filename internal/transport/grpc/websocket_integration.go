package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SteelMount-Captcha-Service/internal/domain"
	"github.com/SteelMount-Captcha-Service/internal/websocket"
	pb "github.com/SteelMount-Captcha-Service/pb/proto/captcha/v1"
)

// WebSocketIntegration handles WebSocket integration for gRPC service
type WebSocketIntegration struct {
	wsService *websocket.WebSocketService
}

// NewWebSocketIntegration creates a new WebSocket integration
func NewWebSocketIntegration(wsService *websocket.WebSocketService) *WebSocketIntegration {
	return &WebSocketIntegration{
		wsService: wsService,
	}
}

// HandleCaptchaEvent handles captcha events from WebSocket
func (wsi *WebSocketIntegration) HandleCaptchaEvent(ctx context.Context, event *websocket.Event) error {
	switch event.Type {
	case "captcha:sendData":
		return wsi.handleCaptchaData(ctx, event)
	case "captcha:request":
		return wsi.handleCaptchaRequest(ctx, event)
	case "captcha:validate":
		return wsi.handleCaptchaValidation(ctx, event)
	default:
		// Unknown event type, ignore
		return nil
	}
}

// handleCaptchaData handles captcha data from client
func (wsi *WebSocketIntegration) handleCaptchaData(ctx context.Context, event *websocket.Event) error {
	// Extract data from event
	data, ok := event.Data["data"].(string)
	if !ok {
		return fmt.Errorf("invalid event data format")
	}
	
	// Parse captcha data
	var captchaData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &captchaData); err != nil {
		return fmt.Errorf("failed to parse captcha data: %w", err)
	}
	
	// Handle different captcha types
	captchaType, ok := captchaData["type"].(string)
	if !ok {
		return fmt.Errorf("captcha type not specified")
	}
	
	switch captchaType {
	case "drag_drop_solution":
		return wsi.handleDragDropSolution(ctx, event, captchaData)
	case "click_solution":
		return wsi.handleClickSolution(ctx, event, captchaData)
	case "swipe_solution":
		return wsi.handleSwipeSolution(ctx, event, captchaData)
	default:
		return fmt.Errorf("unknown captcha type: %s", captchaType)
	}
}

// handleDragDropSolution handles drag and drop solution
func (wsi *WebSocketIntegration) handleDragDropSolution(ctx context.Context, event *websocket.Event, captchaData map[string]interface{}) error {
	// Extract solution data
	solution, ok := captchaData["solution"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid solution format")
	}
	
	captchaID, ok := captchaData["captchaId"].(string)
	if !ok {
		return fmt.Errorf("captcha ID not specified")
	}
	
	// Create validation event
	validationEvent := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:validation_result",
		Data: map[string]interface{}{
			"captcha_id": captchaID,
			"solution":   solution,
			"status":     "received",
		},
		Timestamp: time.Now(),
		ClientID:  event.ClientID,
	}
	
	// Send validation event back to client
	return wsi.wsService.BroadcastToClient(event.ClientID, validationEvent)
}

// handleClickSolution handles click solution
func (wsi *WebSocketIntegration) handleClickSolution(ctx context.Context, event *websocket.Event, captchaData map[string]interface{}) error {
	// Similar to drag drop solution
	return wsi.handleDragDropSolution(ctx, event, captchaData)
}

// handleSwipeSolution handles swipe solution
func (wsi *WebSocketIntegration) handleSwipeSolution(ctx context.Context, event *websocket.Event, captchaData map[string]interface{}) error {
	// Similar to drag drop solution
	return wsi.handleDragDropSolution(ctx, event, captchaData)
}

// handleCaptchaRequest handles captcha request from client
func (wsi *WebSocketIntegration) handleCaptchaRequest(ctx context.Context, event *websocket.Event) error {
	// Extract request data
	requestData, ok := event.Data["request"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid request data format")
	}
	
	// Create new challenge request
	req := &pb.NewChallengeRequest{
		ClientId:  event.ClientID,
		ChallengeType: pb.ChallengeType_CHALLENGE_TYPE_DRAG_DROP, // Default type
		Complexity:    50, // Default complexity
	}
	
	// Set challenge type if specified
	if challengeType, ok := requestData["challenge_type"].(string); ok {
		switch challengeType {
		case "drag_drop":
			req.ChallengeType = pb.ChallengeType_CHALLENGE_TYPE_DRAG_DROP
		case "click":
			req.ChallengeType = pb.ChallengeType_CHALLENGE_TYPE_CLICK
		case "swipe":
			req.ChallengeType = pb.ChallengeType_CHALLENGE_TYPE_SWIPE
		}
	}
	
	// Set complexity if specified
	if complexity, ok := requestData["complexity"].(float64); ok {
		req.Complexity = int32(complexity)
	}
	
	// Send challenge request event
	challengeEvent := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:challenge_request",
		Data: map[string]interface{}{
			"request": req,
		},
		Timestamp: time.Now(),
		ClientID:  event.ClientID,
	}
	
	return wsi.wsService.BroadcastToClient(event.ClientID, challengeEvent)
}

// handleCaptchaValidation handles captcha validation
func (wsi *WebSocketIntegration) handleCaptchaValidation(ctx context.Context, event *websocket.Event) error {
	// Extract validation data
	validationData, ok := event.Data["validation"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid validation data format")
	}
	
	// Create validation result event
	resultEvent := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:validation_result",
		Data: map[string]interface{}{
			"validation": validationData,
			"status":     "processed",
		},
		Timestamp: time.Now(),
		ClientID:  event.ClientID,
	}
	
	return wsi.wsService.BroadcastToClient(event.ClientID, resultEvent)
}

// SendChallengeToClient sends a challenge to a specific client
func (wsi *WebSocketIntegration) SendChallengeToClient(clientID string, challenge *domain.Challenge) error {
	// Create challenge event
	event := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:challenge",
		Data: map[string]interface{}{
			"challenge_id": challenge.ID,
			"type":         challenge.Type.String(),
			"html":         challenge.HTML,
			"instructions": challenge.Instructions,
		},
		Timestamp: time.Now(),
		ClientID:  clientID,
	}
	
	return wsi.wsService.BroadcastToClient(clientID, event)
}

// SendValidationResult sends validation result to client
func (wsi *WebSocketIntegration) SendValidationResult(clientID string, result *domain.ChallengeResult) error {
	// Create validation result event
	event := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:validation_result",
		Data: map[string]interface{}{
			"challenge_id": result.ChallengeID,
			"valid":        result.Valid,
			"score":        result.Score,
			"message":      result.Message,
		},
		Timestamp: time.Now(),
		ClientID:  clientID,
	}
	
	return wsi.wsService.BroadcastToClient(clientID, event)
}

// SendError sends an error message to client
func (wsi *WebSocketIntegration) SendError(clientID string, err error) error {
	// Create error event
	event := &websocket.Event{
		ID:   uuid.New().String(),
		Type: "captcha:error",
		Data: map[string]interface{}{
			"error": err.Error(),
		},
		Timestamp: time.Now(),
		ClientID:  clientID,
	}
	
	return wsi.wsService.BroadcastToClient(clientID, event)
}

// RegisterEventHandlers registers WebSocket event handlers
func (wsi *WebSocketIntegration) RegisterEventHandlers() {
	// Register captcha event handler
	wsi.wsService.RegisterHandler("captcha:sendData", wsi.HandleCaptchaEvent)
	wsi.wsService.RegisterHandler("captcha:request", wsi.HandleCaptchaEvent)
	wsi.wsService.RegisterHandler("captcha:validate", wsi.HandleCaptchaEvent)
}
