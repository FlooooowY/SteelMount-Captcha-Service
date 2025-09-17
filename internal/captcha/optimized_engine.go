package captcha

import (
	"sync"
	"time"
)

// OptimizedEngine provides high-performance captcha generation with object pooling
type OptimizedEngine struct {
	generators map[string]CaptchaGenerator
	mu         sync.RWMutex
	
	// Object pools for memory efficiency
	dragDropPool *sync.Pool
	clickPool    *sync.Pool
	swipePool    *sync.Pool
	
	// Performance tracking
	generationCount int64
	lastReset       time.Time
	muStats         sync.RWMutex
}

// NewOptimizedEngine creates a new optimized captcha engine
func NewOptimizedEngine() *OptimizedEngine {
	engine := &OptimizedEngine{
		generators: make(map[string]CaptchaGenerator),
		lastReset:  time.Now(),
	}
	
	// Initialize object pools
	engine.dragDropPool = &sync.Pool{
		New: func() interface{} {
			return &DragDropCaptcha{
				Objects: make([]DragObject, 0, 8),
				Targets: make([]DropTarget, 0, 8),
			}
		},
	}
	
	engine.clickPool = &sync.Pool{
		New: func() interface{} {
			return &ClickCaptcha{
				ClickTargets: make([]ClickTarget, 0, 5),
			}
		},
	}
	
	engine.swipePool = &sync.Pool{
		New: func() interface{} {
			return &SwipeCaptcha{
				SwipeTargets: make([]SwipeTarget, 0, 3),
			}
		},
	}
	
	return engine
}

// RegisterCaptcha registers a captcha generator
func (oe *OptimizedEngine) RegisterCaptcha(captchaType string, generator CaptchaGenerator) {
	oe.mu.Lock()
	defer oe.mu.Unlock()
	oe.generators[captchaType] = generator
}

// GenerateChallenge generates a captcha challenge with optimizations
func (oe *OptimizedEngine) GenerateChallenge(captchaType string, complexity int32) (interface{}, interface{}, error) {
	oe.mu.RLock()
	generator, exists := oe.generators[captchaType]
	oe.mu.RUnlock()
	
	if !exists {
		return nil, nil, ErrUnknownCaptchaType
	}
	
	// Update performance stats
	oe.updateStats()
	
	// Generate challenge using the registered generator
	challenge, answer, err := generator.Generate(complexity)
	if err != nil {
		return nil, nil, err
	}
	
	// Optimize based on captcha type
	switch captchaType {
	case "drag_drop":
		return oe.optimizeDragDrop(challenge, answer)
	case "click":
		return oe.optimizeClick(challenge, answer)
	case "swipe":
		return oe.optimizeSwipe(challenge, answer)
	default:
		return challenge, answer, nil
	}
}

// optimizeDragDrop optimizes drag and drop captcha
func (oe *OptimizedEngine) optimizeDragDrop(challenge, answer interface{}) (interface{}, interface{}, error) {
	// Get from pool
	captcha := oe.dragDropPool.Get().(*DragDropCaptcha)
	
	// Reset and populate
	if dragDrop, ok := challenge.(*DragDropCaptcha); ok {
		*captcha = *dragDrop
		// Clear slices to avoid memory leaks
		captcha.Objects = captcha.Objects[:0]
		captcha.Targets = captcha.Targets[:0]
		// Copy data
		captcha.Objects = append(captcha.Objects, dragDrop.Objects...)
		captcha.Targets = append(captcha.Targets, dragDrop.Targets...)
	}
	
	return captcha, answer, nil
}

// optimizeClick optimizes click captcha
func (oe *OptimizedEngine) optimizeClick(challenge, answer interface{}) (interface{}, interface{}, error) {
	// Get from pool
	captcha := oe.clickPool.Get().(*ClickCaptcha)
	
	// Reset and populate
	if click, ok := challenge.(*ClickCaptcha); ok {
		*captcha = *click
		// Clear slices to avoid memory leaks
		captcha.ClickTargets = captcha.ClickTargets[:0]
		// Copy data
		captcha.ClickTargets = append(captcha.ClickTargets, click.ClickTargets...)
	}
	
	return captcha, answer, nil
}

// optimizeSwipe optimizes swipe captcha
func (oe *OptimizedEngine) optimizeSwipe(challenge, answer interface{}) (interface{}, interface{}, error) {
	// Get from pool
	captcha := oe.swipePool.Get().(*SwipeCaptcha)
	
	// Reset and populate
	if swipe, ok := challenge.(*SwipeCaptcha); ok {
		*captcha = *swipe
		// Clear slices to avoid memory leaks
		captcha.SwipeTargets = captcha.SwipeTargets[:0]
		// Copy data
		captcha.SwipeTargets = append(captcha.SwipeTargets, swipe.SwipeTargets...)
	}
	
	return captcha, answer, nil
}

// ReturnToPool returns a captcha to the appropriate pool
func (oe *OptimizedEngine) ReturnToPool(captchaType string, challenge interface{}) {
	switch captchaType {
	case "drag_drop":
		if dragDrop, ok := challenge.(*DragDropCaptcha); ok {
			oe.dragDropPool.Put(dragDrop)
		}
	case "click":
		if click, ok := challenge.(*ClickCaptcha); ok {
			oe.clickPool.Put(click)
		}
	case "swipe":
		if swipe, ok := challenge.(*SwipeCaptcha); ok {
			oe.swipePool.Put(swipe)
		}
	}
}

// updateStats updates performance statistics
func (oe *OptimizedEngine) updateStats() {
	oe.muStats.Lock()
	defer oe.muStats.Unlock()
	
	oe.generationCount++
	
	// Reset counter every minute
	if time.Since(oe.lastReset) > time.Minute {
		oe.generationCount = 1
		oe.lastReset = time.Now()
	}
}

// GetRPS returns current requests per second
func (oe *OptimizedEngine) GetRPS() float64 {
	oe.muStats.RLock()
	defer oe.muStats.RUnlock()
	
	elapsed := time.Since(oe.lastReset).Seconds()
	if elapsed == 0 {
		return 0
	}
	
	return float64(oe.generationCount) / elapsed
}

// GetStats returns engine statistics
func (oe *OptimizedEngine) GetStats() map[string]interface{} {
	oe.muStats.RLock()
	defer oe.muStats.RUnlock()
	
	return map[string]interface{}{
		"generation_count": oe.generationCount,
		"rps":             oe.GetRPS(),
		"uptime_seconds":  time.Since(oe.lastReset).Seconds(),
		"registered_types": len(oe.generators),
	}
}

// CleanupPools cleans up object pools
func (oe *OptimizedEngine) CleanupPools() {
	// Clear pools periodically to prevent memory leaks
	oe.dragDropPool = &sync.Pool{
		New: func() interface{} {
			return &DragDropCaptcha{
				Objects: make([]DragObject, 0, 8),
				Targets: make([]DropTarget, 0, 8),
			}
		},
	}
	
	oe.clickPool = &sync.Pool{
		New: func() interface{} {
			return &ClickCaptcha{
				ClickTargets: make([]ClickTarget, 0, 5),
			}
		},
	}
	
	oe.swipePool = &sync.Pool{
		New: func() interface{} {
			return &SwipeCaptcha{
				SwipeTargets: make([]SwipeTarget, 0, 3),
			}
		},
	}
}
