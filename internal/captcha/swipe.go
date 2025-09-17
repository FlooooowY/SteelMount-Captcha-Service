package captcha

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// SwipeCaptcha represents a swipe-based captcha
type SwipeCaptcha struct {
	ID           string      `json:"id"`
	Image        string      `json:"image"`
	SwipeAreas   []SwipeArea `json:"swipe_areas"`
	Instructions string      `json:"instructions"`
	CanvasWidth  int         `json:"canvas_width"`
	CanvasHeight int         `json:"canvas_height"`
}

// SwipeArea represents a swipeable area
type SwipeArea struct {
	ID        string `json:"id"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Direction string `json:"direction"` // "left", "right", "up", "down"
	Required  bool   `json:"required"`
	Text      string `json:"text,omitempty"`
}

// SwipeGenerator generates swipe-based captchas
type SwipeGenerator struct {
	canvasWidth    int
	canvasHeight   int
	minSwipes      int
	maxSwipes      int
	swipeThreshold int
}

// NewSwipeGenerator creates a new swipe generator
func NewSwipeGenerator(canvasWidth, canvasHeight, minSwipes, maxSwipes, swipeThreshold int) *SwipeGenerator {
	return &SwipeGenerator{
		canvasWidth:    canvasWidth,
		canvasHeight:   canvasHeight,
		minSwipes:      minSwipes,
		maxSwipes:      maxSwipes,
		swipeThreshold: swipeThreshold,
	}
}

// Generate creates a new swipe captcha
func (g *SwipeGenerator) Generate(complexity int32) (*SwipeCaptcha, interface{}, error) {
	// Determine number of swipes based on complexity
	numSwipes := g.calculateSwipeCount(complexity)

	// Generate swipe areas
	swipeAreas, correctSequence := g.generateSwipeAreas(numSwipes)

	// Generate image data
	imageData := g.generateImage(swipeAreas)

	// Create captcha
	captcha := &SwipeCaptcha{
		ID:           fmt.Sprintf("swipe_%d", time.Now().UnixNano()),
		Image:        imageData,
		SwipeAreas:   swipeAreas,
		Instructions: g.generateInstructions(complexity),
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}

	return captcha, correctSequence, nil
}

// GenerateHTML generates HTML for the swipe captcha
func (g *SwipeGenerator) GenerateHTML(captcha *SwipeCaptcha) (string, error) {
	// Convert captcha to JSON for JavaScript
	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		return "", fmt.Errorf("failed to marshal captcha: %w", err)
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Swipe Captcha</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .captcha-container {
            max-width: %dpx;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 20px;
        }
        .instructions {
            text-align: center;
            margin-bottom: 20px;
            font-size: 16px;
            color: #333;
        }
        .canvas {
            position: relative;
            width: %dpx;
            height: %dpx;
            border: 2px solid #ddd;
            border-radius: 4px;
            background: #fafafa;
            margin: 0 auto;
            overflow: hidden;
            touch-action: none;
        }
        .swipe-area {
            position: absolute;
            border: 2px solid #007bff;
            border-radius: 8px;
            background: rgba(0,123,255,0.1);
            cursor: grab;
            user-select: none;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            color: #007bff;
        }
        .swipe-area:hover {
            background: rgba(0,123,255,0.2);
        }
        .swipe-area.dragging {
            cursor: grabbing;
            z-index: 1000;
            box-shadow: 0 4px 15px rgba(0,0,0,0.3);
        }
        .swipe-area.completed {
            border-color: #28a745;
            background: rgba(40,167,69,0.2);
            color: #28a745;
        }
        .swipe-area.incorrect {
            border-color: #dc3545;
            background: rgba(220,53,69,0.2);
            color: #dc3545;
        }
        .submit-btn {
            display: block;
            margin: 20px auto 0;
            padding: 10px 20px;
            background: #007bff;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        .submit-btn:hover {
            background: #0056b3;
        }
        .submit-btn:disabled {
            background: #6c757d;
            cursor: not-allowed;
        }
        .progress {
            text-align: center;
            margin-top: 10px;
            font-size: 14px;
            color: #666;
        }
        .direction-hint {
            position: absolute;
            font-size: 12px;
            color: #666;
            pointer-events: none;
        }
    </style>
</head>
<body>
    <div class="captcha-container">
        <div class="instructions" id="instructions">%s</div>
        <div class="canvas" id="canvas"></div>
        <div class="progress" id="progress">Swipe the areas in the correct direction</div>
        <button class="submit-btn" id="submitBtn" onclick="submitSolution()" disabled>Submit</button>
    </div>

    <script>
        const captchaData = %s;
        let completedSwipes = new Set();
        let solution = [];
        let isDragging = false;
        let dragElement = null;
        let startX = 0;
        let startY = 0;
        
        // Initialize the captcha
        function initCaptcha() {
            const canvas = document.getElementById('canvas');
            
            // Create swipe areas
            captchaData.swipe_areas.forEach(area => {
                const areaEl = document.createElement('div');
                areaEl.className = 'swipe-area';
                areaEl.id = 'area-' + area.id;
                areaEl.style.left = area.x + 'px';
                areaEl.style.top = area.y + 'px';
                areaEl.style.width = area.width + 'px';
                areaEl.style.height = area.height + 'px';
                areaEl.textContent = area.text || 'Swipe';
                areaEl.dataset.areaId = area.id;
                areaEl.dataset.direction = area.direction;
                areaEl.dataset.required = area.required;
                
                // Add direction hint
                const hint = document.createElement('div');
                hint.className = 'direction-hint';
                hint.textContent = getDirectionArrow(area.direction);
                hint.style.left = (area.width / 2 - 10) + 'px';
                hint.style.top = '-20px';
                areaEl.appendChild(hint);
                
                // Add event listeners
                areaEl.addEventListener('mousedown', handleMouseDown);
                areaEl.addEventListener('touchstart', handleTouchStart);
                
                canvas.appendChild(areaEl);
            });
            
            // Add global event listeners
            document.addEventListener('mousemove', handleMouseMove);
            document.addEventListener('mouseup', handleMouseUp);
            document.addEventListener('touchmove', handleTouchMove);
            document.addEventListener('touchend', handleTouchEnd);
            
            updateProgress();
        }
        
        function getDirectionArrow(direction) {
            switch(direction) {
                case 'left': return '←';
                case 'right': return '→';
                case 'up': return '↑';
                case 'down': return '↓';
                default: return '?';
            }
        }
        
        function handleMouseDown(e) {
            e.preventDefault();
            startDrag(e.target, e.clientX, e.clientY);
        }
        
        function handleTouchStart(e) {
            e.preventDefault();
            const touch = e.touches[0];
            startDrag(e.target, touch.clientX, touch.clientY);
        }
        
        function startDrag(element, x, y) {
            if (completedSwipes.has(element.dataset.areaId)) return;
            
            isDragging = true;
            dragElement = element;
            startX = x;
            startY = y;
            element.classList.add('dragging');
        }
        
        function handleMouseMove(e) {
            if (!isDragging) return;
            e.preventDefault();
            handleDrag(e.clientX, e.clientY);
        }
        
        function handleTouchMove(e) {
            if (!isDragging) return;
            e.preventDefault();
            const touch = e.touches[0];
            handleDrag(touch.clientX, touch.clientY);
        }
        
        function handleDrag(x, y) {
            if (!dragElement) return;
            
            const deltaX = x - startX;
            const deltaY = y - startY;
            const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
            
            // Only move if distance is significant
            if (distance > 10) {
                const rect = dragElement.getBoundingClientRect();
                const canvasRect = document.getElementById('canvas').getBoundingClientRect();
                const newX = rect.left - canvasRect.left + deltaX;
                const newY = rect.top - canvasRect.top + deltaY;
                
                dragElement.style.left = Math.max(0, Math.min(newX, %d - dragElement.offsetWidth)) + 'px';
                dragElement.style.top = Math.max(0, Math.min(newY, %d - dragElement.offsetHeight)) + 'px';
            }
        }
        
        function handleMouseUp(e) {
            if (!isDragging) return;
            e.preventDefault();
            endDrag(e.clientX, e.clientY);
        }
        
        function handleTouchEnd(e) {
            if (!isDragging) return;
            e.preventDefault();
            const touch = e.changedTouches[0];
            endDrag(touch.clientX, touch.clientY);
        }
        
        let gameCompleted = false;
        let errorCount = 0;
        const maxErrors = 1;
        
        function endDrag(x, y) {
            if (!dragElement || gameCompleted) return;
            
            const deltaX = x - startX;
            const deltaY = y - startY;
            const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
            
            // Check if swipe is valid
            if (distance > %d) {
                const direction = getSwipeDirection(deltaX, deltaY);
                const expectedDirection = dragElement.dataset.direction;
                
                if (direction === expectedDirection) {
                    // Correct swipe
                    dragElement.classList.remove('dragging');
                    dragElement.classList.add('completed');
                    completedSwipes.add(dragElement.dataset.areaId);
                    solution.push({
                        areaId: dragElement.dataset.areaId,
                        direction: direction,
                        distance: distance
                    });
                    
                    // Disable this area
                    dragElement.style.pointerEvents = 'none';
                    dragElement.style.cursor = 'default';
                    
                    // Check if game is completed
                    const requiredAreas = captchaData.swipe_areas.filter(area => area.required);
                    if (completedSwipes.size >= requiredAreas.length) {
                        gameCompleted = true;
                        // Disable all areas
                        document.querySelectorAll('.swipe-area').forEach(area => {
                            area.style.pointerEvents = 'none';
                            area.style.opacity = '0.8';
                        });
                    }
                } else {
                    // Incorrect swipe
                    errorCount++;
                    dragElement.classList.remove('dragging');
                    dragElement.classList.add('incorrect');
                    
                    // Check if too many errors
                    if (errorCount >= maxErrors) {
                        gameCompleted = true;
                        showErrorMessage('Wrong swipe! Getting new captcha...');
                        // Immediately request new captcha
                        setTimeout(() => {
                            requestNewCaptcha();
                        }, 1000);
                        return;
                    }
                    
                    setTimeout(() => {
                        dragElement.classList.remove('incorrect');
                        // Reset position
                        const originalArea = captchaData.swipe_areas.find(a => a.id === dragElement.dataset.areaId);
                        if (originalArea) {
                            dragElement.style.left = originalArea.x + 'px';
                            dragElement.style.top = originalArea.y + 'px';
                        }
                    }, 1000);
                }
            } else {
                // Not enough distance, reset
                dragElement.classList.remove('dragging');
                const originalArea = captchaData.swipe_areas.find(a => a.id === dragElement.dataset.areaId);
                if (originalArea) {
                    dragElement.style.left = originalArea.x + 'px';
                    dragElement.style.top = originalArea.y + 'px';
                }
            }
            
            isDragging = false;
            dragElement = null;
            updateProgress();
        }
        
        function getSwipeDirection(deltaX, deltaY) {
            const absX = Math.abs(deltaX);
            const absY = Math.abs(deltaY);
            
            if (absX > absY) {
                return deltaX > 0 ? 'right' : 'left';
            } else {
                return deltaY > 0 ? 'down' : 'up';
            }
        }
        
        function showErrorMessage(message) {
            const progress = document.getElementById('progress');
            progress.textContent = message;
            progress.style.color = '#dc3545';
            progress.style.fontWeight = 'bold';
        }
        
        function updateProgress() {
            const requiredAreas = captchaData.swipe_areas.filter(area => area.required);
            const completedRequired = requiredAreas.filter(area => completedSwipes.has(area.id));
            
            const progress = document.getElementById('progress');
            const submitBtn = document.getElementById('submitBtn');
            
            if (gameCompleted && errorCount < maxErrors) {
                progress.textContent = 'Perfect! All swipes completed correctly. You can submit now.';
                progress.style.color = '#28a745';
                submitBtn.disabled = false;
            } else {
                progress.textContent = 'Swipe areas in the correct direction (' + completedRequired.length + '/' + requiredAreas.length + ' completed) | One mistake = new captcha!';
                progress.style.color = '#666';
                submitBtn.disabled = true;
            }
        }
        
        function requestNewCaptcha() {
            // Request new captcha from parent window
            window.top.postMessage({
                type: 'captcha:requestNew',
                data: JSON.stringify({
                    reason: 'too_many_errors',
                    currentCaptchaId: captchaData.id,
                    captchaType: 'swipe'
                })
            }, '*');
        }
        
        function submitSolution() {
            // Send solution to parent window
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'swipe_solution',
                    solution: solution,
                    captchaId: captchaData.id
                })
            }, '*');
        }
        
        // Listen for messages from server
        window.addEventListener('message', function(e) {
            if (e.data && e.data.type === 'captcha:serverData') {
                // Handle server data
                console.log('Received server data:', e.data.data);
                // Process server response if needed
            }
        });
        
        // Initialize when page loads
        document.addEventListener('DOMContentLoaded', initCaptcha);
    </script>
</body>
</html>`,
		g.canvasWidth, g.canvasWidth, g.canvasHeight, captcha.Instructions, string(captchaJSON),
		g.canvasWidth, g.canvasHeight, g.swipeThreshold)

	return html, nil
}

// calculateSwipeCount calculates the number of swipes based on complexity
func (g *SwipeGenerator) calculateSwipeCount(complexity int32) int {
	// More complexity = more swipes required
	baseCount := g.minSwipes
	maxCount := g.maxSwipes

	if complexity < 30 {
		return baseCount
	} else if complexity < 60 {
		return baseCount + 1
	} else if complexity < 80 {
		return baseCount + 2
	} else {
		return maxCount
	}
}

// generateSwipeAreas generates swipe areas for the captcha
func (g *SwipeGenerator) generateSwipeAreas(numSwipes int) ([]SwipeArea, []map[string]interface{}) {
	// Ultra-random seed for infinite variations
	// Ultra-enhanced seed for infinite variations
	seed := time.Now().UnixNano() + 
		int64(rand.Intn(1000000)) + 
		int64(numSwipes*11117) + 
		int64(g.canvasWidth*g.canvasHeight) + 
		int64(time.Now().Nanosecond()) + 
		int64(os.Getpid()*17)
	rand.Seed(seed)

	swipeAreas := make([]SwipeArea, numSwipes)
	correctSequence := make([]map[string]interface{}, 0, numSwipes)

	// Expanded directions for more variety (including diagonal)
	directions := []string{
		"left", "right", "up", "down",
		"up-left", "up-right", "down-left", "down-right",
		"circle-clockwise", "circle-counter", "zigzag-horizontal", "zigzag-vertical",
	}
	
	// Advanced shuffling with multiple passes for maximum randomness
	for pass := 0; pass < 7; pass++ {
		for i := range directions {
			j := rand.Intn(len(directions))
			directions[i], directions[j] = directions[j], directions[i]
		}
	}

	for i := 0; i < numSwipes; i++ {
		// Generate random position
		x := rand.Intn(g.canvasWidth - 100)
		y := rand.Intn(g.canvasHeight - 100)

		// Ensure areas don't overlap
		g.avoidOverlap(x, y, swipeAreas[:i])

		direction := directions[rand.Intn(len(directions))]

		area := SwipeArea{
			ID:        fmt.Sprintf("area_%d", i),
			X:         x,
			Y:         y,
			Width:     80,
			Height:    80,
			Direction: direction,
			Required:  true,
			Text:      fmt.Sprintf("Swipe %s", direction),
		}

		swipeAreas[i] = area
		correctSequence = append(correctSequence, map[string]interface{}{
			"areaId":    area.ID,
			"direction": direction,
		})
	}

	return swipeAreas, correctSequence
}

// avoidOverlap ensures swipe areas don't overlap
func (g *SwipeGenerator) avoidOverlap(x, y int, existingAreas []SwipeArea) {
	maxAttempts := 50
	attempts := 0

	for attempts < maxAttempts {
		overlaps := false

		for _, existing := range existingAreas {
			if g.checkOverlap(x, y, 80, 80, existing.X, existing.Y, existing.Width, existing.Height) {
				overlaps = true
				break
			}
		}

		if !overlaps {
			break
		}

		// Reposition
		x = rand.Intn(g.canvasWidth - 100)
		y = rand.Intn(g.canvasHeight - 100)

		attempts++
	}
}

// checkOverlap checks if two rectangles overlap
func (g *SwipeGenerator) checkOverlap(x1, y1, w1, h1, x2, y2, w2, h2 int) bool {
	return !(x1+w1 < x2 || x2+w2 < x1 || y1+h1 < y2 || y2+h2 < y1)
}

// generateImage generates a simple image
func (g *SwipeGenerator) generateImage(swipeAreas []SwipeArea) string {
	// For now, return a simple base64 encoded image
	return "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48L3N2Zz4="
}

// generateInstructions generates instructions based on complexity
func (g *SwipeGenerator) generateInstructions(complexity int32) string {
	instructions := []string{
		"Swipe each area in the direction shown by the arrow",
		"Drag the elements in the correct direction",
		"Move each item in the direction indicated",
		"Swipe the highlighted areas according to their arrows",
	}

	return instructions[rand.Intn(len(instructions))]
}
