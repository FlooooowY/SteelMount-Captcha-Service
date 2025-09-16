package usecase

import (
	"context"
	"fmt"
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
			ChallengeID:      challengeID,
			Solved:           false,
			ConfidencePercent: 0,
			Error:            "challenge expired",
		}, nil
	}

	// Check if already solved
	if challenge.Solved {
		return &domain.ChallengeResult{
			ChallengeID:      challengeID,
			Solved:           true,
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
		ChallengeID:      challengeID,
		Solved:           isValid,
		ConfidencePercent: confidence,
		TimeToSolve:      time.Since(challenge.CreatedAt).Milliseconds(),
		Attempts:         1, // TODO: Track actual attempts
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

// determineChallengeType determines the challenge type based on complexity
func (u *captchaUsecase) determineChallengeType(complexity int32) domain.ChallengeType {
	if complexity < 30 {
		return domain.ChallengeTypeClick
	} else if complexity < 60 {
		return domain.ChallengeTypeDragDrop
	} else if complexity < 80 {
		return domain.ChallengeTypeSwipe
	} else {
		return domain.ChallengeTypeGame
	}
}


// validateAnswer validates a challenge answer
func (u *captchaUsecase) validateAnswer(challenge *domain.Challenge, answer interface{}) (bool, int32) {
	// Simple validation logic
	// TODO: Implement proper validation based on challenge type
	
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
	// TODO: Implement proper click validation
	return expected == actual, 100
}

// validateDragDropAnswer validates a drag-drop challenge answer
func (u *captchaUsecase) validateDragDropAnswer(expected, actual interface{}) (bool, int32) {
	// TODO: Implement proper drag-drop validation
	return expected == actual, 100
}

// validateSwipeAnswer validates a swipe challenge answer
func (u *captchaUsecase) validateSwipeAnswer(expected, actual interface{}) (bool, int32) {
	// TODO: Implement proper swipe validation
	return expected == actual, 100
}

// validateGameAnswer validates a game challenge answer
func (u *captchaUsecase) validateGameAnswer(expected, actual interface{}) (bool, int32) {
	// TODO: Implement proper game validation
	return expected == actual, 100
}

// processFrontendEvent processes a frontend event
func (u *captchaUsecase) processFrontendEvent(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// TODO: Implement frontend event processing
	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeSendClientData,
		ChallengeID: challenge.ID,
		Data:        []byte("event_processed"),
		Timestamp:   time.Now(),
	}, nil
}

// processConnectionClosed processes a connection closed event
func (u *captchaUsecase) processConnectionClosed(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// TODO: Implement connection closed processing
	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeChallengeResult,
		ChallengeID: challenge.ID,
		Data:        []byte("connection_closed"),
		Timestamp:   time.Now(),
	}, nil
}

// processBalancerEvent processes a balancer event
func (u *captchaUsecase) processBalancerEvent(ctx context.Context, challenge *domain.Challenge, event *domain.Event) (*domain.ServerEvent, error) {
	// TODO: Implement balancer event processing
	return &domain.ServerEvent{
		Type:        domain.ServerEventTypeSendClientData,
		ChallengeID: challenge.ID,
		Data:        []byte("balancer_event_processed"),
		Timestamp:   time.Now(),
	}, nil
}
