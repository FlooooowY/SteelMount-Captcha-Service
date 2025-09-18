package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/captcha"
)

// MockRepository for testing
type MockRepository struct {
	challenges map[string]interface{}
	mu         sync.RWMutex
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		challenges: make(map[string]interface{}),
	}
}

func (m *MockRepository) Create(ctx context.Context, challenge interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Simulate minimal storage overhead
	return nil
}

func (m *MockRepository) Get(ctx context.Context, id string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.challenges[id], nil
}

func (m *MockRepository) GetActiveCount(ctx context.Context) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.challenges)
}

func (m *MockRepository) CleanupExpired(ctx context.Context) error {
	return nil
}

// Test100RPSGeneration tests if the system can generate 100 RPS
func Test100RPSGeneration(t *testing.T) {
	// Create captcha engine
	engine := captcha.NewEngine(400, 300)
	
	// Test parameters
	testDuration := 10 * time.Second
	targetRPS := 100
	expectedRequests := int64(testDuration.Seconds() * float64(targetRPS))
	
	var completedRequests int64
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency = time.Hour
	var mu sync.Mutex
	
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()
	
	// Worker pool for concurrent generation
	numWorkers := 50
	requestChan := make(chan struct{}, 1000)
	
	var wg sync.WaitGroup
	
	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			challengeTypes := []string{"click", "drag_drop", "swipe", "game"}
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-requestChan:
					start := time.Now()
					
					// Generate challenge
					challengeType := challengeTypes[workerID%len(challengeTypes)]
					complexity := int32(50 + (workerID * 5) % 50) // Vary complexity
					
					_, _, err := engine.GenerateChallenge(challengeType, complexity)
					
					latency := time.Since(start)
					
					if err != nil {
						t.Errorf("Worker %d failed to generate challenge: %v", workerID, err)
						return
					}
					
					// Update stats
					mu.Lock()
					atomic.AddInt64(&completedRequests, 1)
					totalLatency += latency
					if latency > maxLatency {
						maxLatency = latency
					}
					if latency < minLatency {
						minLatency = latency
					}
					mu.Unlock()
				}
			}
		}(i)
	}
	
	// Request generator
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				close(requestChan)
				return
			case <-ticker.C:
				select {
				case requestChan <- struct{}{}:
				default:
					// Channel full, skip this request
				}
			}
		}
	}()
	
	// Wait for test completion
	<-ctx.Done()
	wg.Wait()
	
	// Calculate results
	actualRPS := float64(completedRequests) / testDuration.Seconds()
	avgLatency := totalLatency / time.Duration(completedRequests)
	
	t.Logf("Performance Test Results:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Target RPS: %d", targetRPS)
	t.Logf("  Actual RPS: %.2f", actualRPS)
	t.Logf("  Completed Requests: %d / %d", completedRequests, expectedRequests)
	t.Logf("  Average Latency: %v", avgLatency)
	t.Logf("  Min Latency: %v", minLatency)
	t.Logf("  Max Latency: %v", maxLatency)
	
	// Verify performance requirements
	if actualRPS < float64(targetRPS)*0.9 { // Allow 10% tolerance
		t.Errorf("Performance requirement not met: achieved %.2f RPS, expected at least %.2f RPS", 
			actualRPS, float64(targetRPS)*0.9)
	}
	
	if avgLatency > time.Millisecond*100 { // Average latency should be under 100ms
		t.Errorf("Average latency too high: %v (should be < 100ms)", avgLatency)
	}
	
	if maxLatency > time.Millisecond*500 { // Max latency should be under 500ms
		t.Errorf("Max latency too high: %v (should be < 500ms)", maxLatency)
	}
}

// TestMemoryUsage tests memory consumption under load
func TestMemoryUsage(t *testing.T) {
	// Test the engine directly for memory usage
	engine := captcha.NewEngine(400, 300)
	
	// Generate 10k challenges to test memory usage
	const numChallenges = 10000
	challenges := make([]string, numChallenges)
	
	start := time.Now()
	
	for i := 0; i < numChallenges; i++ {
		challengeType := []string{"click", "drag_drop", "swipe", "game"}[i%4]
		complexity := int32(i % 100)
		
		html, _, err := engine.GenerateChallenge(challengeType, complexity)
		if err != nil {
			t.Fatalf("Failed to generate challenge %d: %v", i, err)
		}
		
		challenges[i] = html
		
		// Log progress
		if (i+1)%1000 == 0 {
			t.Logf("Generated %d challenges", i+1)
		}
	}
	
	generationTime := time.Since(start)
	
	// Calculate statistics
	totalSize := 0
	for _, challenge := range challenges {
		totalSize += len(challenge)
	}
	
	avgSize := totalSize / numChallenges
	totalSizeMB := float64(totalSize) / (1024 * 1024)
	
	t.Logf("Memory Usage Test Results:")
	t.Logf("  Challenges Generated: %d", numChallenges)
	t.Logf("  Generation Time: %v", generationTime)
	t.Logf("  Average Challenge Size: %d bytes", avgSize)
	t.Logf("  Total Memory Usage: %.2f MB", totalSizeMB)
	t.Logf("  Generation Rate: %.2f challenges/sec", float64(numChallenges)/generationTime.Seconds())
	
	// Verify memory requirements (≤8GB for 10k challenges according to TZ)
	maxMemoryMB := 8 * 1024 // 8GB in MB
	if totalSizeMB > float64(maxMemoryMB) {
		t.Errorf("Memory usage exceeds limit: %.2f MB > %d MB", totalSizeMB, maxMemoryMB)
	}
	
	// Verify generation rate
	generationRate := float64(numChallenges) / generationTime.Seconds()
	if generationRate < 100 { // Should be able to generate at least 100 challenges per second
		t.Errorf("Generation rate too low: %.2f challenges/sec (should be ≥ 100)", generationRate)
	}
}

// TestConcurrentGeneration tests concurrent challenge generation
func TestConcurrentGeneration(t *testing.T) {
	engine := captcha.NewEngine(400, 300)
	
	numGoroutines := 100
	challengesPerGoroutine := 50
	totalChallenges := numGoroutines * challengesPerGoroutine
	
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	
	start := time.Now()
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < challengesPerGoroutine; j++ {
				challengeType := []string{"click", "drag_drop", "swipe", "game"}[j%4]
				complexity := int32((goroutineID*challengesPerGoroutine + j) % 100)
				
				_, _, err := engine.GenerateChallenge(challengeType, complexity)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, challenge %d failed: %v", goroutineID, j, err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	duration := time.Since(start)
	actualRPS := float64(successCount) / duration.Seconds()
	
	t.Logf("Concurrent Generation Test Results:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Challenges per Goroutine: %d", challengesPerGoroutine)
	t.Logf("  Total Challenges: %d", totalChallenges)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Actual RPS: %.2f", actualRPS)
	
	// Verify no errors occurred
	if errorCount > 0 {
		t.Errorf("Concurrent generation had %d errors", errorCount)
	}
	
	// Verify all challenges were generated
	if successCount != int64(totalChallenges) {
		t.Errorf("Not all challenges were generated: %d/%d", successCount, totalChallenges)
	}
	
	// Verify performance under concurrency
	if actualRPS < 100 {
		t.Errorf("Concurrent RPS too low: %.2f (should be ≥ 100)", actualRPS)
	}
}

// BenchmarkChallengeGeneration benchmarks challenge generation
func BenchmarkChallengeGeneration(b *testing.B) {
	engine := captcha.NewEngine(400, 300)
	challengeTypes := []string{"click", "drag_drop", "swipe", "game"}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		challengeType := challengeTypes[i%len(challengeTypes)]
		complexity := int32(i % 100)
		
		_, _, err := engine.GenerateChallenge(challengeType, complexity)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkParallelGeneration benchmarks parallel challenge generation
func BenchmarkParallelGeneration(b *testing.B) {
	engine := captcha.NewEngine(400, 300)
	challengeTypes := []string{"click", "drag_drop", "swipe", "game"}
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			challengeType := challengeTypes[i%len(challengeTypes)]
			complexity := int32(i % 100)
			
			_, _, err := engine.GenerateChallenge(challengeType, complexity)
			if err != nil {
				b.Fatalf("Parallel benchmark failed: %v", err)
			}
			i++
		}
	})
}
