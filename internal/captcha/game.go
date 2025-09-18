package captcha

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// GameGenerator generates simple game-based captchas
type GameGenerator struct {
	canvasWidth  int
	canvasHeight int
}

// GameCaptcha represents a game-based captcha
type GameCaptcha struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Instructions string      `json:"instructions"`
	GameType     string      `json:"game_type"`
	GameData     interface{} `json:"game_data"`
	CanvasWidth  int         `json:"canvas_width"`
	CanvasHeight int         `json:"canvas_height"`
}

// SnakeGameData represents snake game data
type SnakeGameData struct {
	FoodX      int     `json:"food_x"`
	FoodY      int     `json:"food_y"`
	GridSize   int     `json:"grid_size"`
	TargetFood int     `json:"target_food"`
	Speed      int     `json:"speed"`
	Colors     []string `json:"colors"`
}

// MemoryGameData represents memory game data
type MemoryGameData struct {
	Sequence    []int   `json:"sequence"`
	GridSize    int     `json:"grid_size"`
	Cells       int     `json:"cells"`
	ShowTime    int     `json:"show_time"`
	Colors      []string `json:"colors"`
}

// ReactionGameData represents reaction time game data
type ReactionGameData struct {
	TargetTime  int     `json:"target_time"`
	Tolerance   int     `json:"tolerance"`
	Colors      []string `json:"colors"`
	Instructions string `json:"instructions"`
}

// NewGameGenerator creates a new game generator
func NewGameGenerator(canvasWidth, canvasHeight int) *GameGenerator {
	return &GameGenerator{
		canvasWidth:  canvasWidth,
		canvasHeight: canvasHeight,
	}
}

// Generate generates a game captcha based on complexity
func (g *GameGenerator) Generate(complexity int32) (*GameCaptcha, interface{}, error) {
	gameTypes := []string{"snake", "memory", "reaction"}
	gameType := gameTypes[rand.Intn(len(gameTypes))]
	
	// Adjust game difficulty based on complexity
	switch gameType {
	case "snake":
		return g.generateSnakeGame(complexity)
	case "memory":
		return g.generateMemoryGame(complexity)
	case "reaction":
		return g.generateReactionGame(complexity)
	default:
		return g.generateSnakeGame(complexity)
	}
}

// generateSnakeGame generates a simple snake-like game
func (g *GameGenerator) generateSnakeGame(complexity int32) (*GameCaptcha, interface{}, error) {
	gridSize := 20
	targetFood := 3 + int(complexity/25) // 3-7 food items based on complexity
	
	gameData := &SnakeGameData{
		FoodX:      rand.Intn(g.canvasWidth/gridSize) * gridSize,
		FoodY:      rand.Intn(g.canvasHeight/gridSize) * gridSize,
		GridSize:   gridSize,
		TargetFood: targetFood,
		Speed:      200 - int(complexity*2), // Faster with higher complexity
		Colors:     []string{"#ff6b6b", "#4ecdc4", "#45b7d1", "#f9ca24", "#6c5ce7"},
	}
	
	captcha := &GameCaptcha{
		ID:           fmt.Sprintf("game-%d", time.Now().UnixNano()),
		Type:         "game",
		GameType:     "snake",
		Instructions: fmt.Sprintf("Use arrow keys to collect %d food items. Don't hit the walls!", targetFood),
		GameData:     gameData,
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}
	
	// Expected answer: number of food items collected
	expectedAnswer := map[string]interface{}{
		"type":        "snake_completion",
		"target_food": targetFood,
		"min_score":   targetFood,
	}
	
	return captcha, expectedAnswer, nil
}

// generateMemoryGame generates a memory sequence game
func (g *GameGenerator) generateMemoryGame(complexity int32) (*GameCaptcha, interface{}, error) {
	sequenceLength := 3 + int(complexity/20) // 3-8 sequence length
	gridSize := 4 // 4x4 grid
	
	// Generate random sequence
	sequence := make([]int, sequenceLength)
	for i := range sequence {
		sequence[i] = rand.Intn(gridSize * gridSize)
	}
	
	gameData := &MemoryGameData{
		Sequence:  sequence,
		GridSize:  gridSize,
		Cells:     gridSize * gridSize,
		ShowTime:  2000 - int(complexity*10), // Shorter show time with higher complexity
		Colors:    []string{"#3498db", "#e74c3c", "#2ecc71", "#f39c12", "#9b59b6"},
	}
	
	captcha := &GameCaptcha{
		ID:           fmt.Sprintf("game-%d", time.Now().UnixNano()),
		Type:         "game",
		GameType:     "memory",
		Instructions: fmt.Sprintf("Remember and repeat the sequence of %d highlighted cells", sequenceLength),
		GameData:     gameData,
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}
	
	// Expected answer: the correct sequence
	expectedAnswer := map[string]interface{}{
		"type":     "memory_sequence",
		"sequence": sequence,
	}
	
	return captcha, expectedAnswer, nil
}

// generateReactionGame generates a reaction time game
func (g *GameGenerator) generateReactionGame(complexity int32) (*GameCaptcha, interface{}, error) {
	targetTime := 1000 + rand.Intn(2000) // 1-3 seconds
	tolerance := 300 - int(complexity*2)  // Stricter tolerance with higher complexity
	
	gameData := &ReactionGameData{
		TargetTime:   targetTime,
		Tolerance:    tolerance,
		Colors:       []string{"#e74c3c", "#2ecc71", "#f39c12", "#3498db"},
		Instructions: "Click when the circle turns green!",
	}
	
	captcha := &GameCaptcha{
		ID:           fmt.Sprintf("game-%d", time.Now().UnixNano()),
		Type:         "game",
		GameType:     "reaction",
		Instructions: fmt.Sprintf("Wait for the green signal, then click as fast as possible! Target: %dms", targetTime),
		GameData:     gameData,
		CanvasWidth:  g.canvasWidth,
		CanvasHeight: g.canvasHeight,
	}
	
	// Expected answer: reaction time within tolerance
	expectedAnswer := map[string]interface{}{
		"type":        "reaction_time",
		"target_time": targetTime,
		"tolerance":   tolerance,
	}
	
	return captcha, expectedAnswer, nil
}

// GenerateHTML generates HTML for the game captcha
func (g *GameGenerator) GenerateHTML(captcha *GameCaptcha) (string, error) {
	// Convert captcha to JSON for JavaScript
	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		return "", fmt.Errorf("failed to marshal captcha: %w", err)
	}

	var gameSpecificHTML string
	var gameSpecificJS string

	switch captcha.GameType {
	case "snake":
		gameSpecificHTML, gameSpecificJS = g.generateSnakeHTML()
	case "memory":
		gameSpecificHTML, gameSpecificJS = g.generateMemoryHTML()
	case "reaction":
		gameSpecificHTML, gameSpecificJS = g.generateReactionHTML()
	default:
		gameSpecificHTML, gameSpecificJS = g.generateSnakeHTML()
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Game Captcha</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
            user-select: none;
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
        .game-canvas {
            width: %dpx;
            height: %dpx;
            border: 2px solid #ddd;
            border-radius: 4px;
            background: #fafafa;
            margin: 0 auto;
            display: block;
            cursor: crosshair;
        }
        .game-info {
            text-align: center;
            margin: 10px 0;
            font-size: 14px;
            color: #666;
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
        .game-over {
            position: absolute;
            top: 50%%;
            left: 50%%;
            transform: translate(-50%%, -50%%);
            background: rgba(0,0,0,0.8);
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            display: none;
        }
        %s
    </style>
</head>
<body>
    <div class="captcha-container">
        <div class="instructions" id="instructions">%s</div>
        <canvas class="game-canvas" id="gameCanvas" width="%d" height="%d"></canvas>
        <div class="game-info" id="gameInfo">Game loading...</div>
        %s
        <button class="submit-btn" id="submitBtn" onclick="submitSolution()" disabled>Submit</button>
        <div class="game-over" id="gameOver">
            <h3>Game Complete!</h3>
            <p id="gameResult"></p>
        </div>
    </div>

    <script>
        const captchaData = %s;
        const canvas = document.getElementById('gameCanvas');
        const ctx = canvas.getContext('2d');
        let gameState = {
            completed: false,
            score: 0,
            result: null,
            startTime: Date.now()
        };
        
        // Game-specific variables and functions
        %s
        
        // Common game functions
        function submitSolution() {
            if (!gameState.completed) return;
            
            // Send solution to parent window
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'game_solution',
                    game_type: captchaData.game_type,
                    result: gameState.result,
                    score: gameState.score,
                    captchaId: captchaData.id,
                    completion_time: Date.now() - gameState.startTime
                })
            }, '*');
        }
        
        function completeGame(success, message) {
            gameState.completed = true;
            gameState.result = success;
            
            const gameOver = document.getElementById('gameOver');
            const gameResult = document.getElementById('gameResult');
            const submitBtn = document.getElementById('submitBtn');
            
            gameResult.textContent = message;
            gameOver.style.display = 'block';
            submitBtn.disabled = false;
            
            // Send real-time completion event
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'game_event',
                    event: 'completed',
                    success: success,
                    score: gameState.score,
                    captchaId: captchaData.id
                })
            }, '*');
        }
        
        function updateGameInfo(info) {
            document.getElementById('gameInfo').textContent = info;
        }
        
        // Listen for messages from server
        window.addEventListener('message', function(e) {
            if (e.data && e.data.type === 'captcha:serverData') {
                console.log('Received server data:', e.data.data);
                // Process server response if needed
            }
        });
        
        // Send game events to server
        function sendGameEvent(eventType, data) {
            window.top.postMessage({
                type: 'captcha:sendData',
                data: JSON.stringify({
                    type: 'game_event',
                    event: eventType,
                    data: data,
                    captchaId: captchaData.id,
                    timestamp: Date.now()
                })
            }, '*');
        }
        
        // Initialize game when page loads
        document.addEventListener('DOMContentLoaded', function() {
            initGame();
            
            // Send game start event
            sendGameEvent('started', {
                game_type: captchaData.game_type,
                complexity: captchaData.complexity || 50
            });
        });
    </script>
</body>
</html>`,
		g.canvasWidth, g.canvasWidth, g.canvasHeight, // CSS dimensions
		gameSpecificHTML, // Additional CSS
		captcha.Instructions,
		g.canvasWidth, g.canvasHeight, // Canvas dimensions
		gameSpecificHTML, // Additional HTML elements
		string(captchaJSON),
		gameSpecificJS, // Game-specific JavaScript
	)

	return html, nil
}

// generateSnakeHTML generates HTML and JS for snake game
func (g *GameGenerator) generateSnakeHTML() (string, string) {
	css := `
        .snake-controls {
            text-align: center;
            margin: 10px 0;
            font-size: 12px;
            color: #666;
        }
    `
	
	html := `<div class="snake-controls">Use arrow keys to move</div>`
	
	js := `
        let snake = [{x: 200, y: 200}];
        let direction = {x: 0, y: 0};
        let food = {x: 0, y: 0};
        let foodCollected = 0;
        let gameRunning = false;
        
        function initGame() {
            const gameData = captchaData.game_data;
            food.x = gameData.food_x;
            food.y = gameData.food_y;
            gameRunning = true;
            
            updateGameInfo('Use arrow keys to collect ' + gameData.target_food + ' food items');
            
            // Game loop
            gameLoop();
            
            // Keyboard controls
            document.addEventListener('keydown', function(e) {
                if (!gameRunning) return;
                
                sendGameEvent('keypress', {key: e.code});
                
                switch(e.code) {
                    case 'ArrowUp':
                        if (direction.y === 0) direction = {x: 0, y: -gameData.grid_size};
                        break;
                    case 'ArrowDown':
                        if (direction.y === 0) direction = {x: 0, y: gameData.grid_size};
                        break;
                    case 'ArrowLeft':
                        if (direction.x === 0) direction = {x: -gameData.grid_size, y: 0};
                        break;
                    case 'ArrowRight':
                        if (direction.x === 0) direction = {x: gameData.grid_size, y: 0};
                        break;
                }
                e.preventDefault();
            });
        }
        
        function gameLoop() {
            if (!gameRunning) return;
            
            // Move snake
            const head = {x: snake[0].x + direction.x, y: snake[0].y + direction.y};
            
            // Check wall collision
            if (head.x < 0 || head.x >= canvas.width || head.y < 0 || head.y >= canvas.height) {
                gameRunning = false;
                completeGame(false, 'Game Over! Hit the wall.');
                return;
            }
            
            // Check self collision
            if (snake.some(segment => segment.x === head.x && segment.y === head.y)) {
                gameRunning = false;
                completeGame(false, 'Game Over! Hit yourself.');
                return;
            }
            
            snake.unshift(head);
            
            // Check food collision
            if (head.x === food.x && head.y === food.y) {
                foodCollected++;
                gameState.score = foodCollected;
                
                sendGameEvent('food_collected', {count: foodCollected});
                
                if (foodCollected >= captchaData.game_data.target_food) {
                    gameRunning = false;
                    completeGame(true, 'Success! Collected all food items.');
                    return;
                } else {
                    // Generate new food
                    const gameData = captchaData.game_data;
                    food.x = Math.floor(Math.random() * (canvas.width / gameData.grid_size)) * gameData.grid_size;
                    food.y = Math.floor(Math.random() * (canvas.height / gameData.grid_size)) * gameData.grid_size;
                }
            } else {
                snake.pop();
            }
            
            // Draw game
            draw();
            
            updateGameInfo('Collected: ' + foodCollected + '/' + captchaData.game_data.target_food);
            
            setTimeout(gameLoop, captchaData.game_data.speed);
        }
        
        function draw() {
            // Clear canvas
            ctx.fillStyle = '#fafafa';
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            
            // Draw snake
            ctx.fillStyle = '#4ecdc4';
            snake.forEach(segment => {
                ctx.fillRect(segment.x, segment.y, captchaData.game_data.grid_size - 2, captchaData.game_data.grid_size - 2);
            });
            
            // Draw food
            ctx.fillStyle = '#ff6b6b';
            ctx.fillRect(food.x, food.y, captchaData.game_data.grid_size - 2, captchaData.game_data.grid_size - 2);
        }
    `
	
	return css + html, js
}

// generateMemoryHTML generates HTML and JS for memory game
func (g *GameGenerator) generateMemoryHTML() (string, string) {
	css := `
        .memory-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 5px;
            max-width: 200px;
            margin: 10px auto;
        }
        .memory-cell {
            aspect-ratio: 1;
            border: 2px solid #ddd;
            border-radius: 4px;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        .memory-cell:hover {
            border-color: #007bff;
        }
        .memory-cell.active {
            background-color: #007bff;
            border-color: #007bff;
        }
        .memory-cell.correct {
            background-color: #28a745;
            border-color: #28a745;
        }
        .memory-cell.incorrect {
            background-color: #dc3545;
            border-color: #dc3545;
        }
    `
	
	html := `<div class="memory-grid" id="memoryGrid"></div>`
	
	js := `
        let sequence = [];
        let userSequence = [];
        let currentStep = 0;
        let showingSequence = false;
        
        function initGame() {
            const gameData = captchaData.game_data;
            sequence = gameData.sequence;
            
            createGrid();
            setTimeout(() => {
                showSequence();
            }, 1000);
        }
        
        function createGrid() {
            const grid = document.getElementById('memoryGrid');
            const gameData = captchaData.game_data;
            
            for (let i = 0; i < gameData.cells; i++) {
                const cell = document.createElement('div');
                cell.className = 'memory-cell';
                cell.dataset.index = i;
                cell.onclick = () => cellClicked(i);
                grid.appendChild(cell);
            }
        }
        
        function showSequence() {
            showingSequence = true;
            updateGameInfo('Watch the sequence...');
            
            let index = 0;
            const showNext = () => {
                if (index < sequence.length) {
                    const cellIndex = sequence[index];
                    const cell = document.querySelector('[data-index="' + cellIndex + '"]');
                    cell.classList.add('active');
                    
                    setTimeout(() => {
                        cell.classList.remove('active');
                        index++;
                        setTimeout(showNext, 200);
                    }, 500);
                } else {
                    showingSequence = false;
                    updateGameInfo('Now repeat the sequence by clicking the cells');
                }
            };
            
            showNext();
        }
        
        function cellClicked(index) {
            if (showingSequence || gameState.completed) return;
            
            sendGameEvent('cell_clicked', {index: index, step: currentStep});
            
            const cell = document.querySelector('[data-index="' + index + '"]');
            userSequence.push(index);
            
            if (sequence[currentStep] === index) {
                // Correct
                cell.classList.add('correct');
                currentStep++;
                
                if (currentStep >= sequence.length) {
                    // Sequence completed
                    gameState.score = sequence.length;
                    setTimeout(() => {
                        completeGame(true, 'Perfect! Sequence completed correctly.');
                    }, 500);
                }
            } else {
                // Incorrect
                cell.classList.add('incorrect');
                setTimeout(() => {
                    completeGame(false, 'Wrong sequence! Try again.');
                }, 500);
            }
        }
    `
	
	return css + html, js
}

// generateReactionHTML generates HTML and JS for reaction game
func (g *GameGenerator) generateReactionHTML() (string, string) {
	css := `
        .reaction-circle {
            width: 200px;
            height: 200px;
            border-radius: 50%;
            margin: 20px auto;
            cursor: pointer;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            font-weight: bold;
            color: white;
        }
    `
	
	html := `<div class="reaction-circle" id="reactionCircle">Wait...</div>`
	
	js := `
        let reactionStartTime = 0;
        let waitingForReaction = false;
        let gameStarted = false;
        
        function initGame() {
            const circle = document.getElementById('reactionCircle');
            const gameData = captchaData.game_data;
            
            circle.style.backgroundColor = '#dc3545';
            circle.textContent = 'Click to start';
            
            circle.onclick = function() {
                if (!gameStarted) {
                    startReactionTest();
                } else if (waitingForReaction) {
                    handleReaction();
                }
            };
        }
        
        function startReactionTest() {
            gameStarted = true;
            const circle = document.getElementById('reactionCircle');
            const gameData = captchaData.game_data;
            
            circle.style.backgroundColor = '#ffc107';
            circle.textContent = 'Wait for green...';
            
            updateGameInfo('Wait for the circle to turn green, then click as fast as possible!');
            
            // Random delay before showing green
            const delay = 2000 + Math.random() * 3000;
            setTimeout(() => {
                circle.style.backgroundColor = '#28a745';
                circle.textContent = 'CLICK NOW!';
                reactionStartTime = Date.now();
                waitingForReaction = true;
                
                sendGameEvent('green_shown', {timestamp: reactionStartTime});
            }, delay);
        }
        
        function handleReaction() {
            if (!waitingForReaction) return;
            
            const reactionTime = Date.now() - reactionStartTime;
            const gameData = captchaData.game_data;
            const circle = document.getElementById('reactionCircle');
            
            waitingForReaction = false;
            gameState.score = reactionTime;
            
            sendGameEvent('reaction_clicked', {reaction_time: reactionTime});
            
            const isWithinTolerance = Math.abs(reactionTime - gameData.target_time) <= gameData.tolerance;
            
            if (reactionTime < 150) {
                // Too fast, probably cheating
                circle.style.backgroundColor = '#dc3545';
                circle.textContent = 'Too fast!';
                completeGame(false, 'Reaction too fast! You clicked before the signal.');
            } else if (isWithinTolerance) {
                // Good reaction time
                circle.style.backgroundColor = '#28a745';
                circle.textContent = reactionTime + 'ms';
                completeGame(true, 'Great reaction time! (' + reactionTime + 'ms)');
            } else {
                // Too slow or too fast
                circle.style.backgroundColor = '#ffc107';
                circle.textContent = reactionTime + 'ms';
                completeGame(false, 'Reaction time: ' + reactionTime + 'ms (target: ~' + gameData.target_time + 'ms)');
            }
        }
    `
	
	return css + html, js
}
