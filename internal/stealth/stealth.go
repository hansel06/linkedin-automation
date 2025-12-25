package stealth

// Package stealth implements human-like interaction behaviors (mouse, typing, timing, scrolling).

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/hansel06/linkedin-automation/internal/browser"
	"github.com/hansel06/linkedin-automation/internal/config"
	"github.com/hansel06/linkedin-automation/internal/logging"
)

// StealthEngine provides human-like behavior simulation for browser automation.
// All browser interactions should go through StealthEngine, not directly through BrowserManager.
// It implements mandatory anti-detection techniques:
// - Human-like mouse movement (Bezier curves, overshoot, corrections)
// - Randomized timing (delays, think time, scroll speed)
// - Browser fingerprint masking (via BrowserManager)
// Plus additional techniques: random scrolling, realistic typing, hover, scheduling, rate limiting.
type StealthEngine struct {
	browser *browser.BrowserManager // Browser manager for low-level operations
	config  *config.StealthConfig   // Timing, speeds from config
	logger  *logging.Logger         // For logging stealth operations
	rng     *rand.Rand              // Random number generator (seeded with time for true randomness)
}

// NewStealthEngine creates a new stealth engine instance.
// The random number generator is seeded with the current time for true randomness (non-reproducible).
// For debugging, a seed can be provided via config if needed.
func NewStealthEngine(bm *browser.BrowserManager, cfg *config.StealthConfig, logger *logging.Logger) *StealthEngine {
	if bm == nil || cfg == nil || logger == nil {
		// Return nil if dependencies are missing (caller should check)
		return nil
	}

	return &StealthEngine{
		browser: bm,
		config:  cfg,
		logger:  logger,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())), // Time-based seed for true randomness
	}
}

// MoveMouse moves the mouse from (fromX, fromY) to (toX, toY) using a Bezier curve.
// This creates human-like mouse movement that avoids detection.
//
// Algorithm:
// 1. Generate control points for Bezier curve (creates curved path, not straight)
// 2. Divide curve into variable number of steps based on distance
// 3. Calculate position at each step with variable speed (accelerate at start, decelerate at end)
// 4. Add small random offsets at each step (micro-corrections)
// 5. Add overshoot: go slightly past target, then correct back
// 6. Execute movement using Rod's mouse API
//
// This implements the mandatory "human-like mouse movement" anti-detection technique.
func (se *StealthEngine) MoveMouse(page *browser.Page, fromX, fromY, toX, toY int) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	// Calculate distance
	dx := float64(toX - fromX)
	dy := float64(toY - fromY)
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 1 {
		// Already at target
		return nil
	}

	// Generate control points for Bezier curve (20-40% offset from start/end)
	offsetFactor1 := 0.2 + se.rng.Float64()*0.2 // 20-40%
	offsetFactor2 := 0.2 + se.rng.Float64()*0.2

	// Control point 1: random offset from start
	cp1X := float64(fromX) + dx*offsetFactor1 + (se.rng.Float64()-0.5)*distance*0.1
	cp1Y := float64(fromY) + dy*offsetFactor1 + (se.rng.Float64()-0.5)*distance*0.1

	// Control point 2: random offset from end
	cp2X := float64(toX) - dx*offsetFactor2 + (se.rng.Float64()-0.5)*distance*0.1
	cp2Y := float64(toY) - dy*offsetFactor2 + (se.rng.Float64()-0.5)*distance*0.1

	p0 := Point{X: float64(fromX), Y: float64(fromY)}
	p1 := Point{X: cp1X, Y: cp1Y}
	p2 := Point{X: cp2X, Y: cp2Y}
	p3 := Point{X: float64(toX), Y: float64(toY)}

	// Calculate number of steps (10-50 based on distance)
	steps := 10 + int(distance/20)
	if steps > 50 {
		steps = 50
	}
	if steps < 10 {
		steps = 10
	}

	// Move along Bezier curve
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		eased := smoothstep(t) // Apply easing for speed variation

		// Calculate point on curve
		point := bezierPoint(eased, p0, p1, p2, p3)

		// Add small random jitter (±1-3 pixels)
		jitterX := (se.rng.Float64() - 0.5) * 3
		jitterY := (se.rng.Float64() - 0.5) * 3

		x := int(point.X + jitterX)
		y := int(point.Y + jitterY)

		rodPage.Mouse.MoveTo(proto.NewPoint(float64(x), float64(y)))
		
		// Update visible mouse tracker on page (if function exists) - use TryEval to avoid errors
		_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, x, y))
		
		time.Sleep(randomDuration(2*time.Millisecond, 8*time.Millisecond, se.rng))
	}

	// Overshoot: go slightly past target, then correct back
	overshootDist := 5 + se.rng.Intn(6) // 5-10 pixels
	overshootX := toX + int(float64(overshootDist)*(dx/distance))
	overshootY := toY + int(float64(overshootDist)*(dy/distance))

	rodPage.Mouse.MoveTo(proto.NewPoint(float64(overshootX), float64(overshootY)))
	_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, overshootX, overshootY))
	time.Sleep(randomDuration(10*time.Millisecond, 30*time.Millisecond, se.rng))

	// Correct back to exact target
	rodPage.Mouse.MoveTo(proto.NewPoint(float64(toX), float64(toY)))
	_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, toX, toY))

	se.logger.Debug().
		Int("from_x", fromX).Int("from_y", fromY).
		Int("to_x", toX).Int("to_y", toY).
		Int("steps", steps).
		Msg("Mouse moved with Bezier curve")

	return nil
}

// Click clicks an element with human-like mouse movement.
// It first moves the mouse to the element using Bezier curves, then clicks.
// Includes "thinking time" (random pause) before the click.
//
// This combines mouse movement and timing techniques for natural behavior.
func (se *StealthEngine) Click(page *browser.Page, selector string) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	se.logger.Debug().Str("selector", selector).Msg("Clicking element")

	// Get element
	element, err := rodPage.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s: %w", selector, err)
	}

	// Get element position using bounding box
	box, err := element.Shape()
	if err != nil {
		return fmt.Errorf("failed to get element shape: %w", err)
	}

	// Calculate center from bounding box (use first quad point)
	if len(box.Quads) == 0 || len(box.Quads[0]) < 8 {
		return fmt.Errorf("invalid element shape")
	}

	// Quads format: [x1, y1, x2, y2, x3, y3, x4, y4]
	quad := box.Quads[0]
	minX := quad[0]
	maxX := quad[0]
	minY := quad[1]
	maxY := quad[1]

	for i := 0; i < 4; i++ {
		x := quad[i*2]
		y := quad[i*2+1]
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	// Calculate center with small random offset (±2-5 pixels)
	centerX := int((minX + maxX) / 2)
	centerY := int((minY + maxY) / 2)
	offsetX := randomInt(-5, 5, se.rng)
	offsetY := randomInt(-5, 5, se.rng)
	targetX := centerX + offsetX
	targetY := centerY + offsetY

	// Get current mouse position (or use viewport center as start)
	// For simplicity, start from a point near the element
	startX := centerX - randomInt(50, 150, se.rng)
	startY := centerY - randomInt(50, 150, se.rng)

	// Move mouse to element using Bezier curve
	// Update visible tracker at start
	_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, startX, startY))
	
	if err := se.MoveMouse(page, startX, startY, targetX, targetY); err != nil {
		return fmt.Errorf("failed to move mouse: %w", err)
	}

	// Thinking time before click
	if err := se.WaitHumanized(); err != nil {
		return err
	}

	// Perform click
	rodPage.Mouse.Click(proto.InputMouseButtonLeft, 1)

	// Small pause after click
	time.Sleep(randomDuration(50*time.Millisecond, 150*time.Millisecond, se.rng))

	se.logger.Debug().Str("selector", selector).Msg("Element clicked")
	return nil
}

// Type types text into an element with human-like keystroke timing.
// This creates realistic typing patterns that avoid detection.
//
// Algorithm:
// 1. Split text into individual characters
// 2. For each character:
//    a. Wait random duration between typing_speed_min and typing_speed_max
//    b. Type character
//    c. Small chance (3%) of typo: type wrong char, wait, backspace, type correct char
// 3. Occasional longer pause (simulating reading/thinking)
//
// This implements the "realistic typing" additional anti-detection technique.
func (se *StealthEngine) Type(page *browser.Page, selector string, text string) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	se.logger.Debug().Str("selector", selector).Int("text_length", len(text)).Msg("Typing text")

	// Get element and focus it
	element, err := rodPage.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s: %w", selector, err)
	}

	if err := element.Focus(); err != nil {
		return fmt.Errorf("failed to focus element: %w", err)
	}

	// Simple adjacent key mapping for typos (common keys)
	adjacentKeys := map[rune]rune{
		'q': 'w', 'w': 'q', 'e': 'w', 'r': 'e', 't': 'r', 'y': 't', 'u': 'y', 'i': 'u', 'o': 'i', 'p': 'o',
		'a': 's', 's': 'a', 'd': 's', 'f': 'd', 'g': 'f', 'h': 'g', 'j': 'h', 'k': 'j', 'l': 'k',
		'z': 'x', 'x': 'z', 'c': 'x', 'v': 'c', 'b': 'v', 'n': 'b', 'm': 'n',
	}

	// Type each character using the element reference directly
	for i, char := range text {
		// Random typing speed
		time.Sleep(randomDuration(se.config.TypingSpeedMin, se.config.TypingSpeedMax, se.rng))

		// 3% chance of typo
		if se.rng.Float64() < 0.03 && adjacentKeys[char] != 0 {
			// Type wrong character using JavaScript
			wrongChar := adjacentKeys[char]
			// Escape special characters for JavaScript
			charStr := escapeJSChar(string(wrongChar))
			_, err := element.Eval(fmt.Sprintf(`this.value = (this.value || '') + '%s'`, charStr))
			if err != nil {
				return fmt.Errorf("failed to type character: %w", err)
			}
			time.Sleep(randomDuration(100*time.Millisecond, 300*time.Millisecond, se.rng))

			// Backspace
			_, err = element.Eval(`this.value = (this.value || '').slice(0, -1)`)
			if err != nil {
				return fmt.Errorf("failed to backspace: %w", err)
			}
			time.Sleep(randomDuration(50*time.Millisecond, 150*time.Millisecond, se.rng))

			// Type correct character
			charStr = escapeJSChar(string(char))
			_, err = element.Eval(fmt.Sprintf(`this.value = (this.value || '') + '%s'`, charStr))
			if err != nil {
				return fmt.Errorf("failed to type character: %w", err)
			}
		} else {
			// Type correct character using JavaScript
			charStr := escapeJSChar(string(char))
			_, err := element.Eval(fmt.Sprintf(`this.value = (this.value || '') + '%s'`, charStr))
			if err != nil {
				return fmt.Errorf("failed to type character: %w", err)
			}
		}

		// Occasional reading pause (every 5-10 characters)
		if i > 0 && (i+1)%randomInt(5, 10, se.rng) == 0 {
			time.Sleep(randomDuration(200*time.Millisecond, 500*time.Millisecond, se.rng))
		}
	}

	// Final pause after typing
	time.Sleep(randomDuration(100*time.Millisecond, 300*time.Millisecond, se.rng))

	se.logger.Debug().Str("selector", selector).Msg("Text typed")
	return nil
}

// Scroll scrolls the page with variable speed and occasional scroll-backs.
// This creates natural scrolling behavior.
//
// Algorithm:
// 1. Determine scroll direction and distance
// 2. Use variable scroll speed (from config range)
// 3. Accelerate at start, decelerate at end
// 4. Occasional scroll-back: scroll up a bit, then continue down (simulates reading)
//
// This implements the "random scrolling" additional anti-detection technique.
func (se *StealthEngine) Scroll(page *browser.Page, direction string, distance int) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	// Determine scroll direction
	scrollDelta := distance
	if direction == "up" {
		scrollDelta = -distance
	}

	se.logger.Debug().
		Str("direction", direction).
		Int("distance", distance).
		Msg("Scrolling page")

	// Calculate number of steps based on distance
	steps := 10
	if abs(distance) > 100 {
		steps = 20
	}
	if abs(distance) > 200 {
		steps = 30
	}

	// Variable scroll speed with easing
	scrollPerStep := float64(scrollDelta) / float64(steps)
	totalScrolled := 0

	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		eased := smoothstep(t)

		// Calculate scroll amount for this step
		currentScroll := int(scrollPerStep * eased)
		if i > 0 {
			prevEased := smoothstep(float64(i-1) / float64(steps-1))
			currentScroll = int(scrollPerStep * (eased - prevEased))
		}

		// Random speed variation within config range
		speed := randomInt(se.config.ScrollSpeedMin, se.config.ScrollSpeedMax, se.rng)
		actualScroll := int(float64(currentScroll) * float64(speed) / 200.0) // Normalize

		if actualScroll != 0 {
			rodPage.Mouse.Scroll(0, 0, actualScroll)
			totalScrolled += actualScroll
		}

		time.Sleep(randomDuration(10*time.Millisecond, 30*time.Millisecond, se.rng))
	}

	// 20% chance of scroll-back after scrolling 200+ pixels
	if abs(totalScrolled) > 200 && se.rng.Float64() < 0.2 {
		scrollBack := randomInt(20, 50, se.rng)
		rodPage.Mouse.Scroll(0, 0, -scrollBack)
		time.Sleep(randomDuration(200*time.Millisecond, 500*time.Millisecond, se.rng))
		rodPage.Mouse.Scroll(0, 0, scrollBack)
	}

	se.logger.Debug().Msg("Scroll completed")
	return nil
}

// RandomDelay waits for a random duration between min_delay and max_delay from config.
// This is used between actions to create human-like timing patterns.
//
// This implements part of the mandatory "randomized timing" anti-detection technique.
func (se *StealthEngine) RandomDelay() error {
	duration := randomDuration(se.config.MinDelay, se.config.MaxDelay, se.rng)
	se.logger.Debug().Dur("delay", duration).Msg("Random delay")
	time.Sleep(duration)
	return nil
}

// WaitHumanized waits with "thinking time" - a random pause before an action.
// This simulates a human thinking before clicking or typing.
// Typical duration: 200ms to 1 second.
//
// This implements part of the mandatory "randomized timing" anti-detection technique.
func (se *StealthEngine) WaitHumanized() error {
	// Random thinking time between 200ms and 1000ms
	duration := randomDuration(200*time.Millisecond, 1000*time.Millisecond, se.rng)
	se.logger.Debug().Dur("thinking_time", duration).Msg("Humanized wait (thinking)")
	time.Sleep(duration)
	return nil
}

// Hover hovers over an element with random mouse movement.
// The mouse may wander slightly before settling on the element.
// This creates natural hover behavior before clicks.
//
// This implements the "hover/wander" additional anti-detection technique.
func (se *StealthEngine) Hover(page *browser.Page, selector string) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	se.logger.Debug().Str("selector", selector).Msg("Hovering over element")

	// Get element
	element, err := rodPage.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s: %w", selector, err)
	}

	// Get element position using bounding box
	box, err := element.Shape()
	if err != nil {
		return fmt.Errorf("failed to get element shape: %w", err)
	}

	// Calculate center from bounding box (use first quad point)
	if len(box.Quads) == 0 || len(box.Quads[0]) < 8 {
		return fmt.Errorf("invalid element shape")
	}

	// Quads format: [x1, y1, x2, y2, x3, y3, x4, y4]
	quad := box.Quads[0]
	minX := quad[0]
	maxX := quad[0]
	minY := quad[1]
	maxY := quad[1]

	for i := 0; i < 4; i++ {
		x := quad[i*2]
		y := quad[i*2+1]
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	centerX := int((minX + maxX) / 2)
	centerY := int((minY + maxY) / 2)

	// Start near element (not exact center)
	startX := centerX - randomInt(30, 60, se.rng)
	startY := centerY - randomInt(30, 60, se.rng)

	// Move mouse near element
	rodPage.Mouse.MoveTo(proto.NewPoint(float64(startX), float64(startY)))
	_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, startX, startY))
	time.Sleep(randomDuration(50*time.Millisecond, 150*time.Millisecond, se.rng))

	// Wander: 1-3 small random movements
	wanderCount := randomInt(1, 3, se.rng)
	currentX := startX
	currentY := startY

	for i := 0; i < wanderCount; i++ {
		// Random offset (±5-10 pixels)
		offsetX := randomInt(-10, 10, se.rng)
		offsetY := randomInt(-10, 10, se.rng)

		wanderX := currentX + offsetX
		wanderY := currentY + offsetY

		rodPage.Mouse.MoveTo(proto.NewPoint(float64(wanderX), float64(wanderY)))
		_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, wanderX, wanderY))
		time.Sleep(randomDuration(50*time.Millisecond, 150*time.Millisecond, se.rng))

		// Move back toward center
		currentX = wanderX + (centerX-wanderX)/2
		currentY = wanderY + (centerY-wanderY)/2
		rodPage.Mouse.MoveTo(proto.NewPoint(float64(currentX), float64(currentY)))
		_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, currentX, currentY))
		time.Sleep(randomDuration(50*time.Millisecond, 150*time.Millisecond, se.rng))
	}

	// Final movement to exact center
	rodPage.Mouse.MoveTo(proto.NewPoint(float64(centerX), float64(centerY)))
	_, _ = rodPage.Eval(fmt.Sprintf(`(function() { try { if (typeof window.updateMousePosition === 'function') { window.updateMousePosition(%d, %d); } } catch(e) {} })()`, centerX, centerY))
	time.Sleep(randomDuration(100*time.Millisecond, 300*time.Millisecond, se.rng))

	se.logger.Debug().Str("selector", selector).Msg("Hover completed")
	return nil
}

// RandomScroll performs a random scroll action (forward or back).
// It randomly decides whether to scroll forward, back, or do a scroll-back pattern.
// This adds natural noise to scrolling behavior.
//
// This implements the "random scrolling" additional anti-detection technique.
func (se *StealthEngine) RandomScroll(page *browser.Page) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	rodPage := page.RodPage()
	if rodPage == nil {
		return fmt.Errorf("page is closed")
	}

	// Randomly decide action: 60% down, 20% up, 20% scroll-back pattern
	action := se.rng.Float64()
	distance := randomInt(50, 300, se.rng)

	se.logger.Debug().
		Float64("action_prob", action).
		Int("distance", distance).
		Msg("Random scroll")

	if action < 0.6 {
		// 60%: Scroll down (normal)
		return se.scrollDown(rodPage, distance)
	} else if action < 0.8 {
		// 20%: Scroll up (back)
		return se.scrollDown(rodPage, -distance)
	} else {
		// 20%: Scroll-back pattern (down, then up, then down)
		if err := se.scrollDown(rodPage, distance/2); err != nil {
			return err
		}
		time.Sleep(randomDuration(200*time.Millisecond, 500*time.Millisecond, se.rng))
		if err := se.scrollDown(rodPage, -distance/3); err != nil {
			return err
		}
		time.Sleep(randomDuration(200*time.Millisecond, 500*time.Millisecond, se.rng))
		return se.scrollDown(rodPage, distance/2)
	}
}

// Helper types and functions for Bezier curves and utilities

// Point represents a 2D coordinate.
type Point struct {
	X float64
	Y float64
}

// randomDuration generates a random duration between min and max.
func randomDuration(min, max time.Duration, rng *rand.Rand) time.Duration {
	if min >= max {
		return min
	}
	diff := max - min
	return min + time.Duration(float64(diff)*rng.Float64())
}

// randomInt generates a random integer between min and max (inclusive).
func randomInt(min, max int, rng *rand.Rand) int {
	if min >= max {
		return min
	}
	return min + rng.Intn(max-min+1)
}

// smoothstep implements an ease-in-out easing function.
// Returns a value between 0 and 1 for t between 0 and 1.
func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// bezierPoint calculates a point on a cubic Bezier curve at parameter t (0.0 to 1.0).
// Uses the formula: B(t) = (1-t)³P₀ + 3(1-t)²tP₁ + 3(1-t)t²P₂ + t³P₃
func bezierPoint(t float64, p0, p1, p2, p3 Point) Point {
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt
	t2 := t * t
	t3 := t2 * t

	return Point{
		X: mt3*p0.X + 3*mt2*t*p1.X + 3*mt*t2*p2.X + t3*p3.X,
		Y: mt3*p0.Y + 3*mt2*t*p1.Y + 3*mt*t2*p2.Y + t3*p3.Y,
	}
}

// scrollDown performs a scroll action with variable speed.
func (se *StealthEngine) scrollDown(rodPage *rod.Page, distance int) error {
	if distance == 0 {
		return nil
	}

	steps := 10
	if abs(distance) > 100 {
		steps = 20
	}
	if abs(distance) > 200 {
		steps = 30
	}

	stepSize := float64(distance) / float64(steps)

	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		eased := smoothstep(t)
		currentStep := stepSize * eased
		if i > 0 {
			currentStep = stepSize * (eased - smoothstep(float64(i-1)/float64(steps-1)))
		}

		rodPage.Mouse.Scroll(0, 0, int(currentStep))
		time.Sleep(randomDuration(10*time.Millisecond, 30*time.Millisecond, se.rng))
	}

	return nil
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// escapeJSChar escapes special characters for JavaScript string literals.
func escapeJSChar(s string) string {
	result := ""
	for _, r := range s {
		switch r {
		case '\'':
			result += "\\'"
		case '"':
			result += "\\\""
		case '\\':
			result += "\\\\"
		case '\n':
			result += "\\n"
		case '\r':
			result += "\\r"
		case '\t':
			result += "\\t"
		default:
			result += string(r)
		}
	}
	return result
}
