package repository

import (
	"context"
	"sync"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/domain"
)

// ChallengeRepository defines the interface for challenge storage
type ChallengeRepository interface {
	Create(ctx context.Context, challenge *domain.Challenge) error
	Get(ctx context.Context, id string) (*domain.Challenge, error)
	Update(ctx context.Context, challenge *domain.Challenge) error
	Delete(ctx context.Context, id string) error
	GetActiveCount(ctx context.Context) int
	CleanupExpired(ctx context.Context) error
}

// InMemoryChallengeRepository implements ChallengeRepository using in-memory storage
type InMemoryChallengeRepository struct {
	challenges map[string]*domain.Challenge
	mu         sync.RWMutex
}

// NewInMemoryChallengeRepository creates a new in-memory challenge repository
func NewInMemoryChallengeRepository() *InMemoryChallengeRepository {
	return &InMemoryChallengeRepository{
		challenges: make(map[string]*domain.Challenge),
	}
}

// Create stores a new challenge
func (r *InMemoryChallengeRepository) Create(ctx context.Context, challenge *domain.Challenge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.challenges[challenge.ID] = challenge
	return nil
}

// Get retrieves a challenge by ID
func (r *InMemoryChallengeRepository) Get(ctx context.Context, id string) (*domain.Challenge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	challenge, exists := r.challenges[id]
	if !exists {
		return nil, ErrChallengeNotFound
	}
	
	return challenge, nil
}

// Update updates an existing challenge
func (r *InMemoryChallengeRepository) Update(ctx context.Context, challenge *domain.Challenge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.challenges[challenge.ID]; !exists {
		return ErrChallengeNotFound
	}
	
	r.challenges[challenge.ID] = challenge
	return nil
}

// Delete removes a challenge by ID
func (r *InMemoryChallengeRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.challenges[id]; !exists {
		return ErrChallengeNotFound
	}
	
	delete(r.challenges, id)
	return nil
}

// GetActiveCount returns the number of active challenges
func (r *InMemoryChallengeRepository) GetActiveCount(ctx context.Context) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	count := 0
	now := time.Now()
	
	for _, challenge := range r.challenges {
		if !challenge.Solved && challenge.ExpiresAt.After(now) {
			count++
		}
	}
	
	return count
}

// CleanupExpired removes expired challenges
func (r *InMemoryChallengeRepository) CleanupExpired(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	
	for id, challenge := range r.challenges {
		if challenge.ExpiresAt.Before(now) {
			delete(r.challenges, id)
		}
	}
	
	return nil
}

// Repository errors
var (
	ErrChallengeNotFound = &RepositoryError{Message: "challenge not found"}
)

// RepositoryError represents a repository error
type RepositoryError struct {
	Message string
}

func (e *RepositoryError) Error() string {
	return e.Message
}
