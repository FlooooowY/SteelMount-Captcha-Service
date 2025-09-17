package captcha

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// DragDropCaptcha represents a drag and drop captcha
type DragDropCaptcha struct {
	ID           string       `json:"id"`
	Objects      []DragObject `json:"objects"`
	Targets      []DropTarget `json:"targets"`
	Instructions string       `json:"instructions"`
	CanvasWidth  int          `json:"canvas_width"`
	CanvasHeight int          `json:"canvas_height"`
}

// DragObject represents a draggable object
type DragObject struct {
	ID            string `json:"id"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Color         string `json:"color"`
	Shape         string `json:"shape"`
	Text          string `json:"text,omitempty"`
	CorrectTarget string `json:"correct_target"`
}

// DropTarget represents a drop target
type DropTarget struct {
	ID     string `json:"id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color"`
	Shape  string `json:"shape"`
	Text   string `json:"text,omitempty"`
}

// DragDropGenerator generates drag and drop captchas
type DragDropGenerator struct {
	canvasWidth  int
	canvasHeight int
	minObjects   int
	maxObjects   int
}

// NewDragDropGenerator creates a new drag and drop generator
func NewDragDropGenerator(canvasWidth, canvasHeight, minObjects, maxObjects int) *DragDropGenerator {
	return &DragDropGenerator{
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
		minObjects:   minObjects,
		maxObjects:   maxObjects,
	}
}

// Generate creates a new drag and drop captcha
func (g *DragDropGenerator) Generate(complexity int32) (*DragDropCaptcha, interface{}, error) {
	// Determine number of objects based on complexity
	numObjects := g.calculateObjectCount(complexity)

	// Generate objects and targets
	objects, targets, correctSequence := g.generateObjectsAndTargets(numObjects)

	// Create captcha
	captcha := &DragDropCaptcha{
		ID:           fmt.Sprintf("dragdrop_%d", time.Now().UnixNano()),
		Objects:      objects,
		Targets:      targets,
		Instructions: g.generateInstructions(complexity),
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}

	return captcha, correctSequence, nil
}

// GenerateHTML generates HTML for the drag and drop captcha
func (g *DragDropGenerator) GenerateHTML(captcha *DragDropCaptcha) (string, error) {
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
    <title>Drag & Drop Captcha</title>
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
        }
        .drag-object {
            position: absolute;
            cursor: move;
            user-select: none;
            border-radius: 4px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            color: white;
            text-shadow: 1px 1px 2px rgba(0,0,0,0.5);
            transition: transform 0.2s ease;
        }
        .drag-object:hover {
            transform: scale(1.05);
        }
        .drag-object.dragging {
            z-index: 1000;
            transform: scale(1.1);
            box-shadow: 0 4px 15px rgba(0,0,0,0.3);
        }
        .drop-target {
            position: absolute;
            border: 2px dashed #ccc;
            border-radius: 4px;
            background: rgba(0,123,255,0.1);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            color: #666;
        }
        .drop-target.drag-over {
            border-color: #007bff;
            background: rgba(0,123,255,0.2);
        }
        .drop-target.correct {
            border-color: #28a745;
            background: rgba(40,167,69,0.2);
        }
        .drop-target.incorrect {
            border-color: #dc3545;
            background: rgba(220,53,69,0.2);
            animation: shake 0.5s;
        }
        .drag-object.correct-drop {
            border: 2px solid #28a745;
            box-shadow: 0 0 15px rgba(40,167,69,0.5);
        }
        .drag-object.incorrect-drop {
            border: 2px solid #dc3545;
            animation: shake 0.5s;
        }
        @keyframes shake {
            0%%, 100%% { transform: translateX(0); }
            25%% { transform: translateX(-5px); }
            75%% { transform: translateX(5px); }
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
    </style>
</head>
<body>
    <div class="captcha-container">
        <div class="instructions" id="instructions">%s</div>
        <div class="canvas" id="canvas"></div>
        <button class="submit-btn" id="submitBtn" onclick="submitSolution()">Submit</button>
    </div>

    <script>
        const captchaData = %s;
        let draggedElement = null;
        let solution = {};
        
        // Initialize the captcha
        function initCaptcha() {
            const canvas = document.getElementById('canvas');
            
            // Create drop targets
            captchaData.targets.forEach(target => {
                const targetEl = document.createElement('div');
                targetEl.className = 'drop-target';
                targetEl.id = 'target-' + target.id;
                targetEl.style.left = target.x + 'px';
                targetEl.style.top = target.y + 'px';
                targetEl.style.width = target.width + 'px';
                targetEl.style.height = target.height + 'px';
                targetEl.textContent = target.text || '';
                canvas.appendChild(targetEl);
            });
            
            // Create drag objects
            captchaData.objects.forEach(obj => {
                const objEl = document.createElement('div');
                objEl.className = 'drag-object';
                objEl.id = 'obj-' + obj.id;
                objEl.style.left = obj.x + 'px';
                objEl.style.top = obj.y + 'px';
                objEl.style.width = obj.width + 'px';
                objEl.style.height = obj.height + 'px';
                objEl.style.backgroundColor = obj.color;
                objEl.textContent = obj.text || '';
                
                // Add drag event listeners
                objEl.draggable = true;
                objEl.addEventListener('dragstart', handleDragStart);
                objEl.addEventListener('dragend', handleDragEnd);
                
                canvas.appendChild(objEl);
            });
            
            // Add drop event listeners to targets
            captchaData.targets.forEach(target => {
                const targetEl = document.getElementById('target-' + target.id);
                targetEl.addEventListener('dragover', handleDragOver);
                targetEl.addEventListener('drop', handleDrop);
                targetEl.addEventListener('dragenter', handleDragEnter);
                targetEl.addEventListener('dragleave', handleDragLeave);
            });
            
            // Initialize progress
            updateProgress();
        }
        
        function handleDragStart(e) {
            draggedElement = e.target;
            e.target.classList.add('dragging');
            e.dataTransfer.effectAllowed = 'move';
        }
        
        function handleDragEnd(e) {
            e.target.classList.remove('dragging');
            draggedElement = null;
        }
        
        function handleDragOver(e) {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';
        }
        
        function handleDragEnter(e) {
            e.target.classList.add('drag-over');
        }
        
        function handleDragLeave(e) {
            e.target.classList.remove('drag-over');
        }
        
        let gameCompleted = false;
        let errorCount = 0;
        const maxErrors = 1;
        
        function handleDrop(e) {
            e.preventDefault();
            e.target.classList.remove('drag-over');
            
            if (draggedElement && !gameCompleted) {
                const targetId = e.target.id.replace('target-', '');
                const objectId = draggedElement.id.replace('obj-', '');
                
                // Find the object and target data
                const objectData = captchaData.objects.find(obj => obj.id === objectId);
                const targetData = captchaData.targets.find(target => target.id === targetId);
                
                if (objectData && targetData) {
                    // Check if this is the correct target
                    const isCorrect = objectData.correct_target === targetId;
                    
                    if (isCorrect) {
                        // Correct drop
                        solution[objectId] = targetId;
                        
                        // Move object to target center
                        const rect = e.target.getBoundingClientRect();
                        const canvasRect = document.getElementById('canvas').getBoundingClientRect();
                        const centerX = rect.left - canvasRect.left + (rect.width / 2) - (draggedElement.offsetWidth / 2);
                        const centerY = rect.top - canvasRect.top + (rect.height / 2) - (draggedElement.offsetHeight / 2);
                        
                        draggedElement.style.left = centerX + 'px';
                        draggedElement.style.top = centerY + 'px';
                        
                        // Mark target and object as correct
                        e.target.classList.add('correct');
                        draggedElement.classList.add('correct-drop');
                        
                        // Disable further dragging of this object
                        draggedElement.draggable = false;
                        draggedElement.style.cursor = 'default';
                        draggedElement.style.pointerEvents = 'none';
                        
                        // Check if game is completed
                        if (Object.keys(solution).length === captchaData.objects.length) {
                            gameCompleted = true;
                            // Disable all remaining objects
                            document.querySelectorAll('.drag-object').forEach(obj => {
                                obj.draggable = false;
                                obj.style.pointerEvents = 'none';
                                obj.style.opacity = '0.8';
                            });
                        }
                    } else {
                        // Incorrect drop
                        errorCount++;
                        e.target.classList.add('incorrect');
                        draggedElement.classList.add('incorrect-drop');
                        
                        // Check if too many errors
                        if (errorCount >= maxErrors) {
                            gameCompleted = true;
                            showErrorMessage('Wrong drop! Getting new captcha...');
                            // Immediately request new captcha
                            setTimeout(() => {
                                requestNewCaptcha();
                            }, 1000);
                            return;
                        }
                        
                        // Reset after animation
                        setTimeout(() => {
                            e.target.classList.remove('incorrect');
                            draggedElement.classList.remove('incorrect-drop');
                            
                            // Return object to original position
                            const originalObj = captchaData.objects.find(obj => obj.id === objectId);
                            if (originalObj) {
                                draggedElement.style.left = originalObj.x + 'px';
                                draggedElement.style.top = originalObj.y + 'px';
                            }
                        }, 1000);
                    }
                }
                
                updateProgress();
            }
        }
        
        function showErrorMessage(message) {
            const progress = document.getElementById('progress') || createProgressElement();
            progress.textContent = message;
            progress.style.color = '#dc3545';
            progress.style.fontWeight = 'bold';
        }
        
        function createProgressElement() {
            const progressDiv = document.createElement('div');
            progressDiv.id = 'progress';
            progressDiv.className = 'progress';
            progressDiv.style.textAlign = 'center';
            progressDiv.style.marginTop = '10px';
            progressDiv.style.fontSize = '14px';
            progressDiv.style.color = '#666';
            const submitBtn = document.getElementById('submitBtn');
            document.querySelector('.captcha-container').insertBefore(progressDiv, submitBtn);
            return progressDiv;
        }
        
        function updateProgress() {
            const totalObjects = captchaData.objects.length;
            const correctlyPlaced = Object.keys(solution).length;
            
            const progress = document.getElementById('progress') || createProgressElement();
            const submitBtn = document.getElementById('submitBtn');
            
            if (gameCompleted && errorCount < maxErrors) {
                progress.textContent = 'Perfect! All objects correctly placed. You can submit now.';
                progress.style.color = '#28a745';
                submitBtn.disabled = false;
            } else {
                progress.textContent = 'Drag objects to matching targets (' + correctlyPlaced + '/' + totalObjects + ' completed) | One mistake = new captcha!';
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
                    captchaType: 'drag_drop'
                })
            }, '*');
        }
        
        function submitSolution() {
            // Send solution to parent window
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'drag_drop_solution',
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
		g.canvasWidth, g.canvasWidth, g.canvasHeight, captcha.Instructions, string(captchaJSON))

	return html, nil
}

// calculateObjectCount calculates the number of objects based on complexity
func (g *DragDropGenerator) calculateObjectCount(complexity int32) int {
	// More complexity = more objects
	baseCount := g.minObjects
	maxCount := g.maxObjects

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

// generateObjectsAndTargets generates objects and targets for the captcha
func (g *DragDropGenerator) generateObjectsAndTargets(numObjects int) ([]DragObject, []DropTarget, map[string]string) {
	// Ultra-random seed for infinite variations
	// Ultra-enhanced seed for infinite variations
	seed := time.Now().UnixNano() + 
		int64(rand.Intn(1000000)) + 
		int64(numObjects*9973) + 
		int64(g.canvasWidth*g.canvasHeight) + 
		int64(time.Now().Nanosecond()) + 
		int64(os.Getpid()*13)
	rand.Seed(seed)

	objects := make([]DragObject, numObjects)
	targets := make([]DropTarget, numObjects)
	correctSequence := make(map[string]string)

	// Generate shapes and colors - randomize each time
	// Massive variety of shapes for infinite combinations
	shapes := []string{
		"circle", "square", "triangle", "diamond", "pentagon", "hexagon", "octagon",
		"star", "heart", "cross", "arrow", "oval", "trapezoid", "rhombus", 
		"parallelogram", "kite", "crescent", "flower", "butterfly", "leaf", 
		"teardrop", "lightning", "cloud", "wave", "spiral", "gear", "shield", 
		"crown", "key", "lock", "house", "tree", "mountain", "sun", "moon",
	}
	
	// Expanded color palette with hex codes
	colors := []string{
		"#ff6b6b", "#4ecdc4", "#45b7d1", "#96ceb4", "#feca57", "#ff9ff3", 
		"#54a0ff", "#5f27cd", "#fd79a8", "#fdcb6e", "#6c5ce7", "#a29bfe",
		"#e17055", "#00b894", "#0984e3", "#6c5ce7", "#fd79a8", "#fdcb6e",
		"#e84393", "#00cec9", "#74b9ff", "#a29bfe", "#fd79a8", "#fdcb6e",
		"#ff7675", "#00b894", "#0984e3", "#6c5ce7", "#e84393", "#00cec9",
		"#fab1a0", "#00b894", "#74b9ff", "#a29bfe", "#fd79a8", "#fdcb6e",
	}

	// Advanced shuffling with multiple passes for maximum randomness
	for pass := 0; pass < 5; pass++ {
		for i := range shapes {
			j := rand.Intn(len(shapes))
			shapes[i], shapes[j] = shapes[j], shapes[i]
		}
		for i := range colors {
			j := rand.Intn(len(colors))
			colors[i], colors[j] = colors[j], colors[i]
		}
	}

	for i := 0; i < numObjects; i++ {
		// Generate target first
		target := DropTarget{
			ID:     fmt.Sprintf("target_%d", i),
			X:      rand.Intn(g.canvasWidth - 60),
			Y:      rand.Intn(g.canvasHeight - 60),
			Width:  60,
			Height: 60,
			Color:  "#e9ecef",
			Shape:  "square",
			Text:   fmt.Sprintf("Drop %d here", i+1),
		}

		// Generate object with correct target
		obj := DragObject{
			ID:            fmt.Sprintf("obj_%d", i),
			X:             rand.Intn(g.canvasWidth - 60),
			Y:             rand.Intn(g.canvasHeight - 60),
			Width:         50,
			Height:        50,
			Color:         colors[rand.Intn(len(colors))],
			Shape:         shapes[rand.Intn(len(shapes))],
			Text:          fmt.Sprintf("%d", i+1),
			CorrectTarget: target.ID,
		}

		// Ensure objects and targets don't overlap
		g.avoidOverlap(&obj, &target, objects, targets)

		objects[i] = obj
		targets[i] = target
		correctSequence[obj.ID] = target.ID
	}

	return objects, targets, correctSequence
}

// avoidOverlap ensures objects and targets don't overlap
func (g *DragDropGenerator) avoidOverlap(obj *DragObject, target *DropTarget, existingObjects []DragObject, existingTargets []DropTarget) {
	maxAttempts := 50
	attempts := 0

	for attempts < maxAttempts {
		overlaps := false

		// Check overlap with existing objects
		for _, existing := range existingObjects {
			if g.checkOverlap(obj.X, obj.Y, obj.Width, obj.Height, existing.X, existing.Y, existing.Width, existing.Height) {
				overlaps = true
				break
			}
		}

		// Check overlap with existing targets
		if !overlaps {
			for _, existing := range existingTargets {
				if g.checkOverlap(target.X, target.Y, target.Width, target.Height, existing.X, existing.Y, existing.Width, existing.Height) {
					overlaps = true
					break
				}
			}
		}

		if !overlaps {
			break
		}

		// Reposition
		obj.X = rand.Intn(g.canvasWidth - obj.Width)
		obj.Y = rand.Intn(g.canvasHeight - obj.Height)
		target.X = rand.Intn(g.canvasWidth - target.Width)
		target.Y = rand.Intn(g.canvasHeight - target.Height)

		attempts++
	}
}

// checkOverlap checks if two rectangles overlap
func (g *DragDropGenerator) checkOverlap(x1, y1, w1, h1, x2, y2, w2, h2 int) bool {
	return !(x1+w1 < x2 || x2+w2 < x1 || y1+h1 < y2 || y2+h2 < y1)
}

// generateInstructions generates instructions based on complexity
func (g *DragDropGenerator) generateInstructions(complexity int32) string {
	instructions := []string{
		"Drag the numbered objects to their correct positions",
		"Match each colored object with its corresponding target",
		"Arrange the objects in the correct order by dragging them",
		"Place each numbered item in its designated drop zone",
	}

	return instructions[rand.Intn(len(instructions))]
}
