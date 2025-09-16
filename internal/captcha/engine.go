package captcha

import (
	"fmt"
	"sync"
	"time"
)

// Engine manages all captcha types and generation
type Engine struct {
	dragDropGenerator *DragDropGenerator
	clickGenerator    *ClickGenerator
	swipeGenerator    *SwipeGenerator
	
	// Performance tracking
	generationCount   int64
	generationTime    time.Duration
	mu                sync.RWMutex
}

// NewEngine creates a new captcha engine
func NewEngine(canvasWidth, canvasHeight int) *Engine {
	return &Engine{
		dragDropGenerator: NewDragDropGenerator(canvasWidth, canvasHeight, 3, 8, 50),
		clickGenerator:    NewClickGenerator(canvasWidth, canvasHeight, 2, 5, 20),
		swipeGenerator:    NewSwipeGenerator(canvasWidth, canvasHeight, 1, 3, 50),
	}
}

// GenerateChallenge generates a captcha challenge based on type and complexity
func (e *Engine) GenerateChallenge(challengeType string, complexity int32) (string, interface{}, error) {
	start := time.Now()
	defer func() {
		e.mu.Lock()
		e.generationCount++
		e.generationTime += time.Since(start)
		e.mu.Unlock()
	}()
	
	switch challengeType {
	case "drag_drop":
		return e.generateDragDrop(complexity)
	case "click":
		return e.generateClick(complexity)
	case "swipe":
		return e.generateSwipe(complexity)
	default:
		return "", nil, fmt.Errorf("unknown challenge type: %s", challengeType)
	}
}

// generateDragDrop generates a drag and drop captcha
func (e *Engine) generateDragDrop(complexity int32) (string, interface{}, error) {
	captcha, answer, err := e.dragDropGenerator.Generate(complexity)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate drag-drop captcha: %w", err)
	}
	
	html, err := e.dragDropGenerator.GenerateHTML(captcha)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate drag-drop HTML: %w", err)
	}
	
	return html, answer, nil
}

// generateClick generates a click captcha
func (e *Engine) generateClick(complexity int32) (string, interface{}, error) {
	captcha, answer, err := e.clickGenerator.Generate(complexity)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate click captcha: %w", err)
	}
	
	html, err := e.clickGenerator.GenerateHTML(captcha)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate click HTML: %w", err)
	}
	
	return html, answer, nil
}

// generateSwipe generates a swipe captcha
func (e *Engine) generateSwipe(complexity int32) (string, interface{}, error) {
	captcha, answer, err := e.swipeGenerator.Generate(complexity)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate swipe captcha: %w", err)
	}
	
	html, err := e.swipeGenerator.GenerateHTML(captcha)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate swipe HTML: %w", err)
	}
	
	return html, answer, nil
}

// GetStats returns engine performance statistics
func (e *Engine) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var avgTime time.Duration
	if e.generationCount > 0 {
		avgTime = e.generationTime / time.Duration(e.generationCount)
	}
	
	return map[string]interface{}{
		"total_generations": e.generationCount,
		"total_time":        e.generationTime.Milliseconds(),
		"average_time_ms":   avgTime.Milliseconds(),
		"rps":               e.calculateRPS(),
	}
}

// calculateRPS calculates requests per second
func (e *Engine) calculateRPS() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.generationTime == 0 {
		return 0
	}
	
	return float64(e.generationCount) / e.generationTime.Seconds()
}

// ResetStats resets performance statistics
func (e *Engine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.generationCount = 0
	e.generationTime = 0
}
