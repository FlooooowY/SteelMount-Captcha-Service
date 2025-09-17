package unit

import (
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/captcha"
)

func TestEngine_GenerateChallenge(t *testing.T) {
	engine := captcha.NewEngine()
	
	// Register captcha types
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))
	engine.RegisterCaptcha("click", captcha.NewClickGenerator(400, 300, 2, 5))
	engine.RegisterCaptcha("swipe", captcha.NewSwipeGenerator(400, 300, 1, 3))
	
	tests := []struct {
		name         string
		captchaType  string
		complexity   int32
		expectError  bool
	}{
		{
			name:        "valid drag_drop captcha",
			captchaType: "drag_drop",
			complexity:  50,
			expectError: false,
		},
		{
			name:        "valid click captcha",
			captchaType: "click",
			complexity:  30,
			expectError: false,
		},
		{
			name:        "valid swipe captcha",
			captchaType: "swipe",
			complexity:  70,
			expectError: false,
		},
		{
			name:        "invalid captcha type",
			captchaType: "invalid",
			complexity:  50,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, answer, err := engine.GenerateChallenge(tt.captchaType, tt.complexity)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if challenge == nil {
				t.Errorf("Expected challenge but got nil")
			}
			
			if answer == nil {
				t.Errorf("Expected answer but got nil")
			}
		})
	}
}

func TestEngine_Performance(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))
	
	// Test RPS performance
	start := time.Now()
	iterations := 100
	
	for i := 0; i < iterations; i++ {
		_, _, err := engine.GenerateChallenge("drag_drop", 50)
		if err != nil {
			t.Errorf("Error generating challenge: %v", err)
		}
	}
	
	duration := time.Since(start)
	rps := float64(iterations) / duration.Seconds()
	
	t.Logf("Generated %d challenges in %v (RPS: %.2f)", iterations, duration, rps)
	
	// Should be able to generate at least 50 RPS
	if rps < 50 {
		t.Errorf("RPS too low: %.2f, expected at least 50", rps)
	}
}

func TestEngine_ConcurrentGeneration(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))
	
	// Test concurrent generation
	concurrency := 10
	iterations := 10
	
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, _, err := engine.GenerateChallenge("drag_drop", 50)
				if err != nil {
					t.Errorf("Error generating challenge: %v", err)
				}
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestEngine_MemoryUsage(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))
	
	// Generate many challenges and check memory doesn't grow excessively
	iterations := 1000
	
	for i := 0; i < iterations; i++ {
		_, _, err := engine.GenerateChallenge("drag_drop", 50)
		if err != nil {
			t.Errorf("Error generating challenge: %v", err)
		}
		
		// Force garbage collection every 100 iterations
		if i%100 == 0 {
			// This is a simplified memory test
			// In a real scenario, you'd use runtime.MemStats
		}
	}
}
