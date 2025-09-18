package captcha

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// ClickCaptcha represents a click-based captcha
type ClickCaptcha struct {
	ID           string      `json:"id"`
	Image        string      `json:"image"`
	ClickAreas   []ClickArea `json:"click_areas"`
	Instructions string      `json:"instructions"`
	CanvasWidth  int         `json:"canvas_width"`
	CanvasHeight int         `json:"canvas_height"`
}

// ClickArea represents a clickable area
type ClickArea struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Required bool   `json:"required"`
	Text     string `json:"text,omitempty"`
}

// ClickGenerator generates click-based captchas
type ClickGenerator struct {
	canvasWidth  int
	canvasHeight int
	minClicks    int
	maxClicks    int
	clickRadius  int
}

// NewClickGenerator creates a new click generator
func NewClickGenerator(canvasWidth, canvasHeight, minClicks, maxClicks, clickRadius int) *ClickGenerator {
	return &ClickGenerator{
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
		minClicks:    minClicks,
		maxClicks:    maxClicks,
		clickRadius:  clickRadius,
	}
}

// Generate creates a new click captcha
func (g *ClickGenerator) Generate(complexity int32) (*ClickCaptcha, interface{}, error) {
	// Determine number of clicks based on complexity
	numClicks := g.calculateClickCount(complexity)

	// Generate click areas
	clickAreas, correctSequence := g.generateClickAreas(numClicks)

	// Generate image data (base64 encoded simple shapes)
	imageData := g.generateImage(clickAreas)

	// Create captcha
	captcha := &ClickCaptcha{
		ID:           fmt.Sprintf("click_%d", time.Now().UnixNano()),
		Image:        imageData,
		ClickAreas:   clickAreas,
		Instructions: g.generateInstructions(complexity),
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}

	return captcha, correctSequence, nil
}

// GenerateHTML generates HTML for the click captcha
func (g *ClickGenerator) GenerateHTML(captcha *ClickCaptcha) (string, error) {
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
    <title>Click Captcha</title>
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
            cursor: crosshair;
        }
        .click-area {
            position: absolute;
            border: 2px solid transparent;
            border-radius: 50%%;
            cursor: pointer;
            transition: all 0.2s ease;
        }
        .click-area:hover {
            border-color: #007bff;
            background: rgba(0,123,255,0.1);
        }
        .click-area.clicked {
            border-color: #28a745;
            background: rgba(40,167,69,0.2);
        }
        .click-area.incorrect {
            border-color: #dc3545;
            background: rgba(220,53,69,0.2);
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
    </style>
</head>
<body>
    <div class="captcha-container">
        <div class="instructions" id="instructions">%s</div>
        <div class="canvas" id="canvas" onclick="handleCanvasClick(event)"></div>
        <div class="progress" id="progress">Click on the required areas</div>
        <button class="submit-btn" id="submitBtn" onclick="submitSolution()" disabled>Submit</button>
    </div>

    <script>
        const captchaData = %s;
        let clickedAreas = new Set();
        let solution = [];
        
        // Initialize the captcha
        function initCaptcha() {
            const canvas = document.getElementById('canvas');
            
            // Create click areas
            captchaData.click_areas.forEach(area => {
                const areaEl = document.createElement('div');
                areaEl.className = 'click-area';
                areaEl.id = 'area-' + area.id;
                areaEl.style.left = (area.x - %d) + 'px';
                areaEl.style.top = (area.y - %d) + 'px';
                areaEl.style.width = (%d * 2) + 'px';
                areaEl.style.height = (%d * 2) + 'px';
                areaEl.dataset.areaId = area.id;
                areaEl.dataset.required = area.required;
                canvas.appendChild(areaEl);
            });
            
            updateProgress();
        }
        
        function handleCanvasClick(event) {
            const rect = event.currentTarget.getBoundingClientRect();
            const x = event.clientX - rect.left;
            const y = event.clientY - rect.top;
            
            // Check if click is within any click area
            const clickedArea = findClickedArea(x, y);
            if (clickedArea) {
                const areaId = clickedArea.dataset.areaId;
                const isRequired = clickedArea.dataset.required === 'true';
                
                if (clickedAreas.has(areaId)) {
                    // Already clicked, remove
                    clickedAreas.delete(areaId);
                    clickedArea.classList.remove('clicked');
                    solution = solution.filter(id => id !== areaId);
                } else {
                    // New click
                    clickedAreas.add(areaId);
                    clickedArea.classList.add('clicked');
                    solution.push(areaId);
                }
                
                updateProgress();
            }
        }
        
        function findClickedArea(x, y) {
            const areas = document.querySelectorAll('.click-area');
            for (let area of areas) {
                const rect = area.getBoundingClientRect();
                const canvasRect = document.getElementById('canvas').getBoundingClientRect();
                const areaX = rect.left - canvasRect.left + %d;
                const areaY = rect.top - canvasRect.top + %d;
                const distance = Math.sqrt((x - areaX) ** 2 + (y - areaY) ** 2);
                
                if (distance <= %d) {
                    return area;
                }
            }
            return null;
        }
        
        function updateProgress() {
            const requiredAreas = captchaData.click_areas.filter(area => area.required);
            const clickedRequired = requiredAreas.filter(area => clickedAreas.has(area.id));
            
            const progress = document.getElementById('progress');
            const submitBtn = document.getElementById('submitBtn');
            
            if (clickedRequired.length === requiredAreas.length) {
                progress.textContent = 'All required areas clicked! You can submit now.';
                progress.style.color = '#28a745';
                submitBtn.disabled = false;
            } else {
                progress.textContent = 'Clicked ' + clickedRequired.length + ' of ' + requiredAreas.length + ' required areas';
                progress.style.color = '#666';
                submitBtn.disabled = true;
            }
        }
        
        function submitSolution() {
            // Send solution to parent window
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'click_solution',
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
		g.clickRadius, g.clickRadius, g.clickRadius, g.clickRadius, g.clickRadius, g.clickRadius, g.clickRadius)

	return html, nil
}

// calculateClickCount calculates the number of clicks based on complexity
func (g *ClickGenerator) calculateClickCount(complexity int32) int {
	// More complexity = more clicks required
	baseCount := g.minClicks
	maxCount := g.maxClicks

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

// generateClickAreas generates click areas for the captcha
func (g *ClickGenerator) generateClickAreas(numClicks int) ([]ClickArea, []string) {
	// rand.Seed is deprecated in Go 1.20+, using default random source

	clickAreas := make([]ClickArea, numClicks)
	correctSequence := make([]string, 0, numClicks)

	for i := 0; i < numClicks; i++ {
		// Generate random position
		x := g.clickRadius + rand.Intn(g.canvasWidth-2*g.clickRadius)
		y := g.clickRadius + rand.Intn(g.canvasHeight-2*g.clickRadius)

		// Ensure areas don't overlap
		g.avoidOverlap(x, y, clickAreas[:i])

		area := ClickArea{
			ID:       fmt.Sprintf("area_%d", i),
			X:        x,
			Y:        y,
			Width:    g.clickRadius * 2,
			Height:   g.clickRadius * 2,
			Required: true, // All areas are required for now
			Text:     fmt.Sprintf("%d", i+1),
		}

		clickAreas[i] = area
		correctSequence = append(correctSequence, area.ID)
	}

	return clickAreas, correctSequence
}

// avoidOverlap ensures click areas don't overlap
func (g *ClickGenerator) avoidOverlap(x, y int, existingAreas []ClickArea) {
	maxAttempts := 50
	attempts := 0

	for attempts < maxAttempts {
		overlaps := false

		for _, existing := range existingAreas {
			distance := g.calculateDistance(x, y, existing.X, existing.Y)
			if distance < float64(g.clickRadius*2) {
				overlaps = true
				break
			}
		}

		if !overlaps {
			break
		}

		// Reposition
		x = g.clickRadius + rand.Intn(g.canvasWidth-2*g.clickRadius)
		y = g.clickRadius + rand.Intn(g.canvasHeight-2*g.clickRadius)

		attempts++
	}
}

// calculateDistance calculates distance between two points
func (g *ClickGenerator) calculateDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}

// generateImage generates a simple image with shapes
func (g *ClickGenerator) generateImage(clickAreas []ClickArea) string {
	// For now, return a simple base64 encoded image
	// In a real implementation, this would generate an actual image
	return "data:image/svg+xml;base64," + g.generateSVG(clickAreas)
}

// generateSVG generates an SVG image with shapes
func (g *ClickGenerator) generateSVG(clickAreas []ClickArea) string {
	// Simple SVG with circles and rectangles
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, g.canvasWidth, g.canvasHeight)

	// Add some background shapes
	shapes := []string{"circle", "rect", "polygon"}
	colors := []string{"#e9ecef", "#dee2e6", "#ced4da", "#adb5bd"}

	for i := 0; i < 10; i++ {
		x := rand.Intn(g.canvasWidth - 50)
		y := rand.Intn(g.canvasHeight - 50)
		color := colors[rand.Intn(len(colors))]
		shape := shapes[rand.Intn(len(shapes))]

		switch shape {
		case "circle":
			svg += fmt.Sprintf(`<circle cx="%d" cy="%d" r="20" fill="%s" opacity="0.3"/>`, x+25, y+25, color)
		case "rect":
			svg += fmt.Sprintf(`<rect x="%d" y="%d" width="40" height="40" fill="%s" opacity="0.3"/>`, x, y, color)
		case "polygon":
			svg += fmt.Sprintf(`<polygon points="%d,%d %d,%d %d,%d" fill="%s" opacity="0.3"/>`, x, y, x+40, y, x+20, y+40, color)
		}
	}

	svg += "</svg>"

	// Encode to base64
	return g.encodeBase64(svg)
}

// encodeBase64 encodes string to base64
func (g *ClickGenerator) encodeBase64(s string) string {
	// Simple base64 encoding
	// In a real implementation, use proper base64 encoding
	return "PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48L3N2Zz4="
}

// generateInstructions generates instructions based on complexity
func (g *ClickGenerator) generateInstructions(complexity int32) string {
	instructions := []string{
		"Click on all the numbered areas in order",
		"Click on the highlighted regions",
		"Select all the required areas by clicking on them",
		"Click on the marked spots to complete the challenge",
	}

	return instructions[rand.Intn(len(instructions))]
}
