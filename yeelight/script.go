package yeelight

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Script represents a parsed script with multiple frames
type Script struct {
	Name   string
	Frames []ColorMatrix
}

// ScriptRunner manages script execution
type ScriptRunner struct {
	yeelight      *Yeelight
	currentScript *Script
	stopChan      chan bool
	mu            sync.Mutex
	isRunning     bool
}

// NewScriptRunner creates a new script runner instance
func NewScriptRunner(yl *Yeelight) *ScriptRunner {
	return &ScriptRunner{
		yeelight: yl,
		stopChan: make(chan bool),
	}
}

// ParseScript reads and parses a script file
func ParseScript(filename string) (*Script, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open script file: %w", err)
	}
	defer file.Close()

	script := &Script{
		Name:   filename,
		Frames: []ColorMatrix{},
	}

	currentMatrix := MakeMatrix("#000000", 25)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	hasContent := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" {
			if hasContent {
				// Empty line means new frame
				script.Frames = append(script.Frames, currentMatrix)
				currentMatrix = MakeMatrix("#000000", 25)
				hasContent = false
			}
			continue
		}

		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse command
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		hasContent = true

		switch cmd {
		case "FILL":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: FILL requires a color", lineNum)
			}
			color, err := parseColor(parts[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			currentMatrix.ReplaceAllHex(color)

		case "CLEAR":
			currentMatrix.ReplaceAllHex("#000000")

		case "PIXEL":
			if len(parts) < 4 {
				return nil, fmt.Errorf("line %d: PIXEL requires x y color", lineNum)
			}
			x, y, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			color, err := parseColor(parts[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			currentMatrix.SetHex(Vector{Row: y, Column: x}, color)

		case "ROW":
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: ROW requires row color", lineNum)
			}
			row, err := strconv.Atoi(parts[1])
			if err != nil || row < 0 || row > 4 {
				return nil, fmt.Errorf("line %d: invalid row number", lineNum)
			}
			color, err := parseColor(parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			for x := 0; x < 5; x++ {
				currentMatrix.SetHex(Vector{Row: row, Column: x}, color)
			}

		case "COL":
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: COL requires column color", lineNum)
			}
			col, err := strconv.Atoi(parts[1])
			if err != nil || col < 0 || col > 4 {
				return nil, fmt.Errorf("line %d: invalid column number", lineNum)
			}
			color, err := parseColor(parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			for y := 0; y < 5; y++ {
				currentMatrix.SetHex(Vector{Row: y, Column: col}, color)
			}

		case "CIRCLE":
			if len(parts) < 5 {
				return nil, fmt.Errorf("line %d: CIRCLE requires x y radius color", lineNum)
			}
			x, y, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			radius, err := strconv.Atoi(parts[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid radius", lineNum)
			}
			color, err := parseColor(parts[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			drawCircle(&currentMatrix, x, y, radius, color)

		case "RING":
			if len(parts) < 5 {
				return nil, fmt.Errorf("line %d: RING requires x y radius color", lineNum)
			}
			x, y, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			radius, err := strconv.Atoi(parts[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid radius", lineNum)
			}
			color, err := parseColor(parts[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			drawRing(&currentMatrix, x, y, radius, color)

		case "RECT":
			if len(parts) < 6 {
				return nil, fmt.Errorf("line %d: RECT requires x1 y1 x2 y2 color", lineNum)
			}
			x1, y1, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			x2, y2, err := parseCoordinates(parts[3], parts[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			color, err := parseColor(parts[5])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			drawRect(&currentMatrix, x1, y1, x2, y2, color)

		case "LINE":
			if len(parts) < 6 {
				return nil, fmt.Errorf("line %d: LINE requires x1 y1 x2 y2 color", lineNum)
			}
			x1, y1, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			x2, y2, err := parseCoordinates(parts[3], parts[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			color, err := parseColor(parts[5])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			drawLine(&currentMatrix, x1, y1, x2, y2, color)

		case "CROSS":
			if len(parts) < 5 {
				return nil, fmt.Errorf("line %d: CROSS requires x y size color", lineNum)
			}
			x, y, err := parseCoordinates(parts[1], parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			size, err := strconv.Atoi(parts[3])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid size", lineNum)
			}
			color, err := parseColor(parts[4])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			drawCross(&currentMatrix, x, y, size, color)

		case "ROTATE":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: ROTATE requires degrees", lineNum)
			}
			degrees, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid degrees", lineNum)
			}
			currentMatrix = currentMatrix.Rotate(degrees)

		case "SHIFT":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: SHIFT requires direction", lineNum)
			}
			direction := strings.ToUpper(parts[1])
			currentMatrix = shiftMatrix(currentMatrix, direction)

		case "DIM":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: DIM requires factor", lineNum)
			}
			factor, err := strconv.ParseFloat(parts[1], 64)
			if err != nil || factor < 0 || factor > 1 {
				return nil, fmt.Errorf("line %d: invalid dim factor (must be 0.0-1.0)", lineNum)
			}
			dimMatrix(&currentMatrix, factor)

		default:
			return nil, fmt.Errorf("line %d: unknown command: %s", lineNum, cmd)
		}
	}

	// Add the last frame if there's content
	if hasContent {
		script.Frames = append(script.Frames, currentMatrix)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading script file: %w", err)
	}

	if len(script.Frames) == 0 {
		return nil, fmt.Errorf("script file is empty or contains no valid commands")
	}

	return script, nil
}

// RunScript executes a script with the given interval and timeout
func (sr *ScriptRunner) RunScript(scriptName string, interval, timeout time.Duration) error {
	sr.mu.Lock()
	if sr.isRunning {
		sr.mu.Unlock()
		return fmt.Errorf("a script is already running")
	}
	sr.isRunning = true
	sr.mu.Unlock()

	// Parse the script
	script, err := ParseScript(scriptName)
	if err != nil {
		sr.mu.Lock()
		sr.isRunning = false
		sr.mu.Unlock()
		return err
	}

	sr.currentScript = script

	// Enable the lamp
	if err := sr.yeelight.SetOn(Options{Smooth: 200}); err != nil {
		sr.mu.Lock()
		sr.isRunning = false
		sr.mu.Unlock()
		return fmt.Errorf("failed to turn on lamp: %w", err)
	}

	// Switch to direct mode to enable LED control
	if err := sr.yeelight.SetDirectMode(); err != nil {
		sr.mu.Lock()
		sr.isRunning = false
		sr.mu.Unlock()
		return fmt.Errorf("failed to set direct mode: %w", err)
	}

	// Run the script
	go sr.runLoop(interval, timeout)

	return nil
}

// StopScript stops the currently running script
func (sr *ScriptRunner) StopScript() error {
	sr.mu.Lock()
	if !sr.isRunning {
		sr.mu.Unlock()
		return fmt.Errorf("no script is running")
	}
	sr.mu.Unlock()

	// Signal stop
	sr.stopChan <- true

	// Wait for the loop to finish
	time.Sleep(100 * time.Millisecond)

	return nil
}

// runLoop is the main animation loop
func (sr *ScriptRunner) runLoop(interval, timeout time.Duration) {
	defer func() {
		sr.mu.Lock()
		sr.isRunning = false
		sr.mu.Unlock()
	}()

	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Always turn off the lamp when the loop ends
	defer func() {
		sr.yeelight.SetOff(Options{Smooth: 200})
	}()

	// If interval is 0, display static (first frame only)
	if interval == 0 {
		matrices := []ColorMatrix{sr.currentScript.Frames[0]}
		if err := sr.yeelight.SetMatrix(matrices); err != nil {
			fmt.Printf("Error setting matrix: %v\n", err)
		}

		// Wait for stop signal or timeout
		select {
		case <-sr.stopChan:
			return
		case <-timeoutChan:
			return
		}
	}

	// Animation loop
	frameIndex := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// Display current frame
		matrices := []ColorMatrix{sr.currentScript.Frames[frameIndex]}
		if err := sr.yeelight.SetMatrix(matrices); err != nil {
			fmt.Printf("Error setting matrix: %v\n", err)
		}

		// Move to next frame
		frameIndex = (frameIndex + 1) % len(sr.currentScript.Frames)

		// Wait for next frame, stop signal, or timeout
		select {
		case <-ticker.C:
			continue
		case <-sr.stopChan:
			return
		case <-timeoutChan:
			return
		}
	}
}

// Helper functions

func parseColor(colorStr string) (string, error) {
	colorStr = strings.ToLower(colorStr)

	// Named colors
	namedColors := map[string]string{
		"red":     "#FF0000",
		"green":   "#00FF00",
		"blue":    "#0000FF",
		"white":   "#FFFFFF",
		"yellow":  "#FFFF00",
		"cyan":    "#00FFFF",
		"magenta": "#FF00FF",
		"orange":  "#FFA500",
		"purple":  "#800080",
		"black":   "#000000",
	}

	if hex, ok := namedColors[colorStr]; ok {
		return hex, nil
	}

	// Hex color
	if strings.HasPrefix(colorStr, "#") {
		if len(colorStr) != 7 {
			return "", fmt.Errorf("invalid hex color: %s", colorStr)
		}
		return colorStr, nil
	}

	// Hex without #
	if len(colorStr) == 6 {
		return "#" + colorStr, nil
	}

	return "", fmt.Errorf("unknown color: %s", colorStr)
}

func parseCoordinates(xStr, yStr string) (int, int, error) {
	x, err := strconv.Atoi(xStr)
	if err != nil || x < 0 || x > 4 {
		return 0, 0, fmt.Errorf("invalid x coordinate: %s", xStr)
	}

	y, err := strconv.Atoi(yStr)
	if err != nil || y < 0 || y > 4 {
		return 0, 0, fmt.Errorf("invalid y coordinate: %s", yStr)
	}

	return x, y, nil
}

func drawCircle(matrix *ColorMatrix, cx, cy, radius int, color string) {
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			dx := x - cx
			dy := y - cy
			distance := math.Sqrt(float64(dx*dx + dy*dy))
			if distance <= float64(radius) {
				matrix.SetHex(Vector{Row: y, Column: x}, color)
			}
		}
	}
}

func drawRing(matrix *ColorMatrix, cx, cy, radius int, color string) {
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			dx := x - cx
			dy := y - cy
			distance := math.Sqrt(float64(dx*dx + dy*dy))
			if math.Abs(distance-float64(radius)) < 0.8 {
				matrix.SetHex(Vector{Row: y, Column: x}, color)
			}
		}
	}
}

func drawRect(matrix *ColorMatrix, x1, y1, x2, y2 int, color string) {
	for y := y1; y <= y2 && y < 5; y++ {
		for x := x1; x <= x2 && x < 5; x++ {
			if x >= 0 && y >= 0 {
				matrix.SetHex(Vector{Row: y, Column: x}, color)
			}
		}
	}
}

func drawLine(matrix *ColorMatrix, x1, y1, x2, y2 int, color string) {
	// Bresenham's line algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		if x1 >= 0 && x1 < 5 && y1 >= 0 && y1 < 5 {
			matrix.SetHex(Vector{Row: y1, Column: x1}, color)
		}

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func drawCross(matrix *ColorMatrix, cx, cy, size int, color string) {
	// Horizontal line
	for i := -size; i <= size; i++ {
		x := cx + i
		if x >= 0 && x < 5 {
			matrix.SetHex(Vector{Row: cy, Column: x}, color)
		}
	}

	// Vertical line
	for i := -size; i <= size; i++ {
		y := cy + i
		if y >= 0 && y < 5 {
			matrix.SetHex(Vector{Row: y, Column: cx}, color)
		}
	}
}

func shiftMatrix(matrix ColorMatrix, direction string) ColorMatrix {
	newMatrix := MakeMatrix("#000000", 25)

	switch direction {
	case "UP":
		for y := 0; y < 4; y++ {
			for x := 0; x < 5; x++ {
				color := matrix.GetColor(Vector{Row: y + 1, Column: x})
				newMatrix.SetColor(Vector{Row: y, Column: x}, color)
			}
		}
	case "DOWN":
		for y := 1; y < 5; y++ {
			for x := 0; x < 5; x++ {
				color := matrix.GetColor(Vector{Row: y - 1, Column: x})
				newMatrix.SetColor(Vector{Row: y, Column: x}, color)
			}
		}
	case "LEFT":
		for y := 0; y < 5; y++ {
			for x := 0; x < 4; x++ {
				color := matrix.GetColor(Vector{Row: y, Column: x + 1})
				newMatrix.SetColor(Vector{Row: y, Column: x}, color)
			}
		}
	case "RIGHT":
		for y := 0; y < 5; y++ {
			for x := 1; x < 5; x++ {
				color := matrix.GetColor(Vector{Row: y, Column: x - 1})
				newMatrix.SetColor(Vector{Row: y, Column: x}, color)
			}
		}
	}

	return newMatrix
}

func dimMatrix(matrix *ColorMatrix, factor float64) {
	for i := range matrix.Colors {
		r, g, b := matrix.Colors[i].ToRGB()
		r = byte(float64(r) * factor)
		g = byte(float64(g) * factor)
		b = byte(float64(b) * factor)
		matrix.Colors[i].RGB(int8(r), int8(g), int8(b))
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}