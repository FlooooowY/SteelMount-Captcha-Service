package captcha

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
        
        function handleDrop(e) {
            e.preventDefault();
            e.target.classList.remove('drag-over');
            
            if (draggedElement) {
                const targetId = e.target.id.replace('target-', '');
                const objectId = draggedElement.id.replace('obj-', '');
                
                // Update solution
                solution[objectId] = targetId;
                
                // Move object to target
                const rect = e.target.getBoundingClientRect();
                const canvasRect = document.getElementById('canvas').getBoundingClientRect();
                draggedElement.style.left = (rect.left - canvasRect.left) + 'px';
                draggedElement.style.top = (rect.top - canvasRect.top) + 'px';
                
                // Mark target as used
                e.target.classList.add('correct');
            }
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
	rand.Seed(time.Now().UnixNano())

	objects := make([]DragObject, numObjects)
	targets := make([]DropTarget, numObjects)
	correctSequence := make(map[string]string)

	// Generate shapes and colors
	shapes := []string{"circle", "square", "triangle", "diamond"}
	colors := []string{"#ff6b6b", "#4ecdc4", "#45b7d1", "#96ceb4", "#feca57", "#ff9ff3", "#54a0ff", "#5f27cd"}

	for i := 0; i < numObjects; i++ {
		// Generate object
		obj := DragObject{
			ID:     fmt.Sprintf("obj_%d", i),
			X:      rand.Intn(g.canvasWidth - 60),
			Y:      rand.Intn(g.canvasHeight - 60),
			Width:  50,
			Height: 50,
			Color:  colors[rand.Intn(len(colors))],
			Shape:  shapes[rand.Intn(len(shapes))],
			Text:   fmt.Sprintf("%d", i+1),
		}

		// Generate corresponding target
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
