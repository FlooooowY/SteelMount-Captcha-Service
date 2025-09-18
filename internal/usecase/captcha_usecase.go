package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/captcha"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/domain"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/repository"
	"github.com/google/uuid"
)

// CaptchaUsecase defines the interface for captcha business logic
type CaptchaUsecase interface {
	CreateChallenge(ctx context.Context, complexity int32) (*domain.Challenge, error)
	ValidateChallenge(ctx context.Context, challengeID string, answer interface{}) (*domain.ChallengeResult, error)
	GetChallenge(ctx context.Context, challengeID string) (*domain.Challenge, error)
	ProcessEvent(ctx context.Context, event *domain.Event) (*domain.ServerEvent, error)
	CleanupExpiredChallenges(ctx context.Context) error
	GetActiveChallengesCount(ctx context.Context) int
}

// captchaUsecase implements CaptchaUsecase
type captchaUsecase struct {
	challengeRepo repository.ChallengeRepository
	config        *Config
	engine        *captcha.Engine
}

// Config represents the usecase configuration
type Config struct {
	MaxActiveChallenges int
	ChallengeTimeout    time.Duration
	CleanupInterval     time.Duration
}

// NewCaptchaUsecase creates a new captcha usecase
func NewCaptchaUsecase(challengeRepo repository.ChallengeRepository, config *Config) CaptchaUsecase {
	return &captchaUsecase{
		challengeRepo: challengeRepo,
		config:        config,
		engine:        captcha.NewEngine(400, 300), // Default canvas size
	}
}

// CreateChallenge creates a new captcha challenge
func (u *captchaUsecase) CreateChallenge(ctx context.Context, complexity int32) (*domain.Challenge, error) {
	// Check if we have too many active challenges
	activeCount := u.challengeRepo.GetActiveCount(ctx)
	if activeCount >= u.config.MaxActiveChallenges {
		return nil, fmt.Errorf("maximum active challenges reached: %d", u.config.MaxActiveChallenges)
	}

	// Generate challenge ID
	challengeID := uuid.New().String()

	// Determine challenge type based on complexity
	challengeType := u.determineChallengeType(complexity)

	// Generate challenge content using engine
	html, answer, err := u.engine.GenerateChallenge(string(challengeType), complexity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge content: %w", err)
	}

	// Create challenge
	challenge := &domain.Challenge{
		ID:         challengeID,
		Type:       challengeType,
		Complexity: complexity,
		HTML:       html,
		Answer:     answer,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(u.config.ChallengeTimeout),
		Solved:     false,
		Metadata:   make(map[string]string),
	}

	// Store challenge
	if err := u.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}

	return challenge, nil
}

// ValidateChallenge validates a challenge answer
func (u *captchaUsecase) ValidateChallenge(ctx context.Context, challengeID string, answer interface{}) (*domain.ChallengeResult, error) {
	// Get challenge
	challenge, err := u.challengeRepo.Get(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	// Check if challenge is expired
	if time.Now().After(challenge.ExpiresAt) {
		return &domain.ChallengeResult{
			ChallengeID:       challengeID,
			Solved:            false,
			ConfidencePercent: 0,
			Error:             "challenge expired",
		}, nil
	}

	// Check if already solved
	if challenge.Solved {
		return &domain.ChallengeResult{
			ChallengeID:       challengeID,
			Solved:            true,
			ConfidencePercent: 100,
		}, nil
	}

	// Validate answer
	isValid, confidence := u.validateAnswer(challenge, answer)

	// Update challenge if solved
	if isValid {
		challenge.Solved = true
		u.challengeRepo.Update(ctx, challenge)
	}

	return &domain.ChallengeResult{
		ChallengeID:       challengeID,
		Solved:            isValid,
		ConfidencePercent: confidence,
		TimeToSolve:       time.Since(challenge.CreatedAt).Milliseconds(),
		Attempts:          1, // First attempt
	}, nil
}

// GetChallenge retrieves a challenge by ID
func (u *captchaUsecase) GetChallenge(ctx context.Context, challengeID string) (*domain.Challenge, error) {
	return u.challengeRepo.Get(ctx, challengeID)
}

// ProcessEvent processes a client event
func (u *captchaUsecase) ProcessEvent(ctx context.Context, event *domain.Event) (*domain.ServerEvent, error) {
	// Get challenge
	challenge, err := u.challengeRepo.Get(ctx, event.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	// Process event based on type
	switch event.Type {
	case domain.EventTypeFrontendEvent:
		return u.processFrontendEvent(ctx, challenge, event)
	case domain.EventTypeConnectionClosed:
		return u.processConnectionClosed(ctx, challenge, event)
	case domain.EventTypeBalancerEvent:
		return u.processBalancerEvent(ctx, challenge, event)
	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}
}

// CleanupExpiredChallenges removes expired challenges
func (u *captchaUsecase) CleanupExpiredChallenges(ctx context.Context) error {
	return u.challengeRepo.CleanupExpired(ctx)
}

// GetActiveChallengesCount returns the number of active challenges
func (u *captchaUsecase) GetActiveChallengesCount(ctx context.Context) int {
	return u.challengeRepo.GetActiveCount(ctx)
}

// determineChallengeType determines the challenge type randomly with complexity influence
func (u *captchaUsecase) determineChallengeType(complexity int32) domain.ChallengeType {
	// Ultra-enhanced seed for maximum randomness
	seed := time.Now().UnixNano() + 
		int64(rand.Intn(1000000)) + 
		int64(complexity*7919) + 
		int64(time.Now().Nanosecond()) + 
		int64(os.Getpid()*23) +
		int64(time.Now().Second()*1000)
	rand.Seed(seed)
	
	// Available challenge types (including game for high complexity)
	challengeTypes := []domain.ChallengeType{
		domain.ChallengeTypeClick,
		domain.ChallengeTypeDragDrop,
		domain.ChallengeTypeSwipe,
		domain.ChallengeTypeGame,
	}
	
	// More balanced weights for better distribution
	weights := make([]int, len(challengeTypes))
	
	// Add some randomness to weights themselves
	randomFactor := rand.Intn(20) - 10 // -10 to +10
	
	if complexity < 30 {
		// Low complexity - slightly favor simple types, no games
		weights[0] = 40 + randomFactor // Click
		weights[1] = 30 + rand.Intn(20) // Drag&Drop
		weights[2] = 30 + rand.Intn(20) // Swipe
		weights[3] = 0 // No games for low complexity
	} else if complexity < 60 {
		// Medium complexity - balanced with occasional games
		weights[0] = 30 + rand.Intn(15) // Click
		weights[1] = 30 + rand.Intn(15) // Drag&Drop
		weights[2] = 30 + rand.Intn(15) // Swipe
		weights[3] = 10 + rand.Intn(10) // Some games
	} else {
		// High complexity - favor games and complex types
		weights[0] = 20 + rand.Intn(15) // Click
		weights[1] = 25 + rand.Intn(15) // Drag&Drop
		weights[2] = 25 + rand.Intn(15) // Swipe
		weights[3] = 30 + rand.Intn(20) // More games for high complexity
	}
	
	// Ensure all weights are positive
	for i := range weights {
		if weights[i] < 1 {
			weights[i] = 1
		}
	}
	
	// Weighted random selection
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}
	
	randomValue := rand.Intn(totalWeight)
	currentWeight := 0
	
	for i, weight := range weights {
		currentWeight += weight
		if randomValue < currentWeight {
			return challengeTypes[i]
		}
	}
	
	// Fallback (should never reach here)
	return challengeTypes[rand.Intn(len(challengeTypes))]
}

// validateAnswer validates a challenge answer
func (u *captchaUsecase) validateAnswer(challenge *domain.Challenge, answer interface{}) (bool, int32) {
	// Validation logic based on challenge type

	switch challenge.Type {
	case domain.ChallengeTypeClick:
		return u.validateClickAnswer(challenge.Answer, answer)
	case domain.ChallengeTypeDragDrop:
		return u.validateDragDropAnswer(challenge.Answer, answer)
	case domain.ChallengeTypeSwipe:
		return u.validateSwipeAnswer(challenge.Answer, answer)
	case domain.ChallengeTypeGame:
		return u.validateGameAnswer(challenge.Answer, answer)
	default:
		return false, 0
	}
}

// validateClickAnswer validates a click challenge answer
func (u *captchaUsecase) validateClickAnswer(expected, actual interface{}) (bool, int32) {
	expectedSequence, ok := expected.([]string)
	if !ok {
		return false, 0
	}

	actualSequence, ok := actual.([]string)
	if !ok {
		return false, 0
	}

	if len(expectedSequence) != len(actualSequence) {
		return false, 20 // Partial credit for wrong length
	}

	correctCount := 0
	for i, expectedID := range expectedSequence {
		if i < len(actualSequence) && actualSequence[i] == expectedID {
			correctCount++
		}
	}

	confidence := int32((correctCount * 100) / len(expectedSequence))
	return correctCount == len(expectedSequence), confidence
}

// validateDragDropAnswer validates a drag-drop challenge answer
func (u *captchaUsecase) validateDragDropAnswer(expected, actual interface{}) (bool, int32) {
	expectedMap, ok := expected.(map[string]string)
	if !ok {
		return false, 0
	}

	actualMap, ok := actual.(map[string]string)
	if !ok {
		return false, 0
	}

	if len(expectedMap) != len(actualMap) {
		return false, 20 // Partial credit for wrong count
	}

	correctCount := 0
	for objectID, expectedTarget := range expectedMap {
		if actualTarget, exists := actualMap[objectID]; exists && actualTarget == expectedTarget {
			correctCount++
		}
	}

	confidence := int32((correctCount * 100) / len(expectedMap))
	return correctCount == len(expectedMap), confidence
}

// validateSwipeAnswer validates a swipe challenge answer
func (u *captchaUsecase) validateSwipeAnswer(expected, actual interface{}) (bool, int32) {
	expectedSequence, ok := expected.([]map[string]interface{})
	if !ok {
		return false, 0
	}

	actualSequence, ok := actual.([]map[string]interface{})
	if !ok {
		return false, 0
	}

	if len(expectedSequence) != len(actualSequence) {
		return false, 20 // Partial credit for wrong count
	}

	correctCount := 0
	for i, expectedSwipe := range expectedSequence {
		if i < len(actualSequence) {
			actualSwipe := actualSequence[i]
			if u.validateSwipeGesture(expectedSwipe, actualSwipe) {
				correctCount++
			}
		}
	}

	confidence := int32((correctCount * 100) / len(expectedSequence))
	return correctCount == len(expectedSequence), confidence
}

// validateSwipeGesture validates a single swipe gesture
func (u *captchaUsecase) validateSwipeGesture(expected, actual map[string]interface{}) bool {
	expectedDirection, ok := expected["direction"].(string)
	if !ok {
		return false
	}

	actualDirection, ok := actual["direction"].(string)
	if !ok {
		return false
	}

	return expectedDirection == actualDirection
}

// validateGameAnswer validates a game challenge answer
func (u *captchaUsecase) validateGameAnswer(expected, actual interface{}) (bool, int32) {
	// Expected is a map with validation criteria
	expectedMap, ok := expected.(map[string]interface{})
	if !ok {
		return false, 0
	}

	// Actual is the game result from client
	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		return false, 0
	}

	// Get the game type to determine validation logic
	gameType, exists := expectedMap["type"]
	if !exists {
		return false, 0
	}

	switch gameType {
	case "snake_completion":
		return u.validateSnakeGame(expectedMap, actualMap)
	case "memory_sequence":
		return u.validateMemoryGame(expectedMap, actualMap)
	case "reaction_time":
		return u.validateReactionGame(expectedMap, actualMap)
	default:
		return false, 0
	}
}

// validateSnakeGame validates snake game completion
func (u *captchaUsecase) validateSnakeGame(expected, actual map[string]interface{}) (bool, int32) {
	targetFood, ok := expected["target_food"].(int)
	if !ok {
		return false, 0
	}

	actualScore, ok := actual["score"].(float64)
	if !ok {
		return false, 0
	}

	success, ok := actual["success"].(bool)
	if !ok {
		return false, 0
	}

	if success && int(actualScore) >= targetFood {
		return true, 100
	}

	// Partial credit based on food collected
	confidence := int32((actualScore / float64(targetFood)) * 80)
	if confidence > 80 {
		confidence = 80
	}
	return false, confidence
}

// validateMemoryGame validates memory sequence game
func (u *captchaUsecase) validateMemoryGame(expected, actual map[string]interface{}) (bool, int32) {
	expectedSequence, ok := expected["sequence"].([]int)
	if !ok {
		return false, 0
	}

	actualSequence, ok := actual["sequence"].([]interface{})
	if !ok {
		return false, 0
	}

	// Convert actual sequence to []int
	actualInts := make([]int, len(actualSequence))
	for i, v := range actualSequence {
		if intVal, ok := v.(float64); ok {
			actualInts[i] = int(intVal)
		} else {
			return false, 0
		}
	}

	if len(expectedSequence) != len(actualInts) {
		return false, 20
	}

	correctCount := 0
	for i, expected := range expectedSequence {
		if i < len(actualInts) && actualInts[i] == expected {
			correctCount++
		}
	}

	confidence := int32((correctCount * 100) / len(expectedSequence))
	return correctCount == len(expectedSequence), confidence
}

// validateReactionGame validates reaction time game
func (u *captchaUsecase) validateReactionGame(expected, actual map[string]interface{}) (bool, int32) {
	targetTime, ok := expected["target_time"].(int)
	if !ok {
		return false, 0
	}

	tolerance, ok := expected["tolerance"].(int)
	if !ok {
		return false, 0
	}

	actualTime, ok := actual["reaction_time"].(float64)
	if !ok {
		return false, 0
	}

	// Check if reaction time is within acceptable range
	diff := int(actualTime) - targetTime
	if diff < 0 {
		diff = -diff
	}

	if diff <= tolerance {
		// Perfect reaction time
		confidence := int32(100 - (diff*50)/tolerance)
		if confidence < 70 {
			confidence = 70
		}
		return true, confidence
	}

	// Too far from target, but give some credit if reasonable
	if int(actualTime) < 150 {
		// Too fast, likely cheating
		return false, 0
	}

	if int(actualTime) > 5000 {
		// Too slow, likely not paying attention
		return false, 10
	}

	// Some credit for reasonable reaction time
	confidence := int32(50 - (diff*30)/1000)
	if confidence < 0 {
		confidence = 0
	}
	return false, confidence
}

// processFrontendEvent processes a frontend event
func (u *captchaUsecase) processFrontendEvent(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// Process different types of frontend events
	var responseData []byte
	var err error

	// Try to parse event data as JSON to understand the event
	var eventData map[string]interface{}
	if len(event.Data) > 0 {
		if parseErr := json.Unmarshal(event.Data, &eventData); parseErr == nil {
			// Handle specific event types
			if eventType, exists := eventData["type"]; exists {
				switch eventType {
				case "mouse_move", "click", "keypress":
					// Track user interaction for bot detection
					responseData = []byte(`{"type":"interaction_tracked","status":"ok"}`)
				case "challenge_attempt":
					// Process challenge attempt
					if answer, exists := eventData["answer"]; exists {
						valid, confidence := u.validateAnswer(challenge, answer)
						result := map[string]interface{}{
							"type":       "challenge_result",
							"valid":      valid,
							"confidence": confidence,
						}
						responseData, err = json.Marshal(result)
						if err != nil {
							return nil, fmt.Errorf("failed to marshal challenge result: %w", err)
						}
					}
				default:
					responseData = []byte(`{"type":"event_acknowledged","status":"ok"}`)
				}
			}
		}
	}

	if responseData == nil {
		responseData = []byte(`{"type":"event_processed","status":"ok"}`)
	}

	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeSendClientData,
		ChallengeID: challenge.ID,
		Data:        responseData,
		Timestamp:   time.Now(),
	}, nil
}

// processConnectionClosed processes a connection closed event
func (u *captchaUsecase) processConnectionClosed(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// When connection closes, we should clean up the challenge and mark it as incomplete
	// This helps prevent memory leaks and provides analytics data

	// Mark challenge as incomplete due to connection loss
	result := map[string]interface{}{
		"type":      "connection_closed",
		"challenge": challenge.ID,
		"reason":    "client_disconnected",
		"timestamp": time.Now().Unix(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection closed result: %w", err)
	}

	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeChallengeResult,
		ChallengeID: challenge.ID,
		Data:        resultData,
		Timestamp:   time.Now(),
	}, nil
}

// processBalancerEvent processes a balancer event
func (u *captchaUsecase) processBalancerEvent(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// Process balancer-specific events like shutdown notifications, health checks, etc.
	var eventData map[string]interface{}
	var responseData []byte

	if len(event.Data) > 0 {
		if parseErr := json.Unmarshal(event.Data, &eventData); parseErr == nil {
			if eventType, exists := eventData["type"]; exists {
				switch eventType {
				case "health_check":
					// Respond with health status
					response := map[string]interface{}{
						"type":      "health_response",
						"status":    "healthy",
						"challenge": challenge.ID,
						"timestamp": time.Now().Unix(),
					}
					var err error
					responseData, err = json.Marshal(response)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal health response: %w", err)
					}
				case "shutdown_notice":
					// Acknowledge shutdown and prepare for graceful termination
					response := map[string]interface{}{
						"type":      "shutdown_ack",
						"challenge": challenge.ID,
						"timestamp": time.Now().Unix(),
					}
					var err error
					responseData, err = json.Marshal(response)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal shutdown ack: %w", err)
					}
				default:
					responseData = []byte(`{"type":"balancer_event_ack","status":"processed"}`)
				}
			}
		}
	}

	if responseData == nil {
		responseData = []byte(`{"type":"balancer_event_processed","status":"ok"}`)
	}

	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeSendClientData,
		ChallengeID: challenge.ID,
		Data:        responseData,
		Timestamp:   time.Now(),
	}, nil
}
