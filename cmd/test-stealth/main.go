package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hansel06/linkedin-automation/internal/browser"
	"github.com/hansel06/linkedin-automation/internal/config"
	"github.com/hansel06/linkedin-automation/internal/logging"
	"github.com/hansel06/linkedin-automation/internal/stealth"
)

func main() {
	fmt.Println("=== Stealth Implementation Test ===")
	fmt.Println()

	// Step 1: Load config
	fmt.Println("Step 1: Loading configuration...")
	cfg := config.NewDefault()
	cfg.Browser.Headless = false // Visible so we can see the behavior
	cfg.Browser.BrowserTimeout = 30 * time.Second
	fmt.Println("✓ Configuration loaded")
	fmt.Println()

	// Step 2: Create logger
	fmt.Println("Step 2: Creating logger...")
	logger := logging.NewLogger(cfg.Logging.LogLevel, cfg.Logging.LogFormat)
	logger = logger.WithComponent("test-stealth")
	fmt.Println("✓ Logger created")
	fmt.Println()

	// Step 3: Create browser manager
	fmt.Println("Step 3: Creating browser manager...")
	bm, err := browser.NewBrowserManager(&cfg.Browser, logger)
	if err != nil {
		fmt.Printf("✗ Failed to create browser manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Browser manager created")
	fmt.Println()

	// Step 4: Start browser
	fmt.Println("Step 4: Starting browser...")
	if err := bm.Start(); err != nil {
		fmt.Printf("✗ Failed to start browser: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Browser started")
	fmt.Println()

	// Ensure browser closes at the end
	defer func() {
		fmt.Println()
		fmt.Println("Final Step: Closing browser...")
		if err := bm.Close(); err != nil {
			fmt.Printf("✗ Failed to close browser: %v\n", err)
		} else {
			fmt.Println("✓ Browser closed")
		}
	}()

	// Step 5: Create page
	fmt.Println("Step 5: Creating page...")
	page, err := bm.NewPage()
	if err != nil {
		fmt.Printf("✗ Failed to create page: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Page created")
	fmt.Println()

	// Step 6: Navigate to test page (use local HTML file for better visibility)
	fmt.Println("Step 6: Navigating to test page...")
	
	// Get absolute path to test HTML file
	wd, _ := os.Getwd()
	testHTMLPath := fmt.Sprintf("file://%s/cmd/test-stealth/test-page.html", wd)
	fmt.Printf("  - Loading test page from: %s\n", testHTMLPath)
	
	if err := page.Navigate(testHTMLPath, 30*time.Second); err != nil {
		fmt.Printf("✗ Failed to navigate: %v\n", err)
		fmt.Println("  - Falling back to YouTube...")
		testURL := "https://youtube.com"
		if err := page.Navigate(testURL, 30*time.Second); err != nil {
			fmt.Printf("✗ Failed to navigate to YouTube: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("✓ Navigation completed")
	fmt.Println()

	// Wait for page to load and ensure browser window is focused
	time.Sleep(3 * time.Second)
	fmt.Println("  - Browser window should be visible now. Please focus the browser window!")
	time.Sleep(2 * time.Second)

	// Step 7: Create stealth engine
	fmt.Println("Step 7: Creating stealth engine...")
	stealthEngine := stealth.NewStealthEngine(bm, &cfg.Stealth, logger)
	if stealthEngine == nil {
		fmt.Printf("✗ Failed to create stealth engine\n")
		os.Exit(1)
	}
	fmt.Println("✓ Stealth engine created")
	fmt.Println()

	// Step 8: Test RandomDelay
	fmt.Println()
	fmt.Println("Step 8: Testing RandomDelay()...")
	start := time.Now()
	if err := stealthEngine.RandomDelay(); err != nil {
		fmt.Printf("✗ RandomDelay failed: %v\n", err)
	} else {
		elapsed := time.Since(start)
		fmt.Printf("✓ RandomDelay completed (waited %v)\n", elapsed)
	}
	fmt.Println()

	// Step 9: Test WaitHumanized
	fmt.Println("Step 9: Testing WaitHumanized()...")
	start = time.Now()
	if err := stealthEngine.WaitHumanized(); err != nil {
		fmt.Printf("✗ WaitHumanized failed: %v\n", err)
	} else {
		elapsed := time.Since(start)
		fmt.Printf("✓ WaitHumanized completed (waited %v)\n", elapsed)
	}
	fmt.Println()

	// Step 10: Test mouse movement (Bezier curve)
	fmt.Println("Step 10: Testing MoveMouse() with Bezier curve...")
	fmt.Println("  - Watch the mouse cursor move in a curved path!")
	fmt.Println("  - Make sure the browser window is focused and visible!")
	fmt.Println("  - Starting in 3 seconds...")
	time.Sleep(3 * time.Second)
	
	// Get viewport center as start point
	startX := cfg.Browser.ViewportWidth / 4
	startY := cfg.Browser.ViewportHeight / 4
	endX := cfg.Browser.ViewportWidth * 3 / 4
	endY := cfg.Browser.ViewportHeight * 3 / 4

	fmt.Printf("  - Moving mouse from (%d, %d) to (%d, %d)\n", startX, startY, endX, endY)
	fmt.Println("  - WATCH FOR THE RED CIRCLE moving on the page!")
	if err := stealthEngine.MoveMouse(page, startX, startY, endX, endY); err != nil {
		fmt.Printf("✗ MoveMouse failed: %v\n", err)
	} else {
		fmt.Println("✓ Mouse moved with Bezier curve")
		fmt.Println("  - Did you see the RED CIRCLE move? Check the status box for position updates!")
	}
	time.Sleep(5 * time.Second) // Longer pause to see the result
	
	// Move again in opposite direction
	fmt.Println("  - Moving mouse back...")
	if err := stealthEngine.MoveMouse(page, endX, endY, startX, startY); err != nil {
		fmt.Printf("✗ MoveMouse failed: %v\n", err)
	} else {
		fmt.Println("✓ Mouse moved back")
	}
	time.Sleep(3 * time.Second)
	fmt.Println()

	// Step 11: Test scrolling
	fmt.Println("Step 11: Testing Scroll() with variable speed...")
	fmt.Println("  - Watch the page scroll with variable speed!")
	fmt.Println("  - Starting in 2 seconds...")
	time.Sleep(2 * time.Second)
	
	fmt.Println("  - WATCH THE PAGE SCROLL DOWN!")
	if err := stealthEngine.Scroll(page, "down", 500); err != nil {
		fmt.Printf("✗ Scroll failed: %v\n", err)
	} else {
		fmt.Println("✓ Scrolling completed (variable speed with easing)")
		fmt.Println("  - Did you see the page scroll smoothly?")
	}
	time.Sleep(3 * time.Second) // Pause to see the scroll
	
	// Scroll back up so we can see the input field
	fmt.Println("  - Scrolling back up to see input field...")
	if err := stealthEngine.Scroll(page, "up", 500); err != nil {
		fmt.Printf("⚠ Scroll up failed: %v\n", err)
	} else {
		fmt.Println("✓ Scrolled back up")
	}
	time.Sleep(2 * time.Second)
	fmt.Println()

	// Step 12: Test random scroll
	fmt.Println("Step 12: Testing RandomScroll()...")
	fmt.Println("  - Watch for random scrolling behavior!")
	fmt.Println("  - Starting in 2 seconds...")
	time.Sleep(2 * time.Second)
	
	if err := stealthEngine.RandomScroll(page); err != nil {
		fmt.Printf("✗ RandomScroll failed: %v\n", err)
	} else {
		fmt.Println("✓ Random scroll completed")
		fmt.Println("  - Did you see random scroll movements?")
	}
	time.Sleep(3 * time.Second)
	
	// Scroll back to top for typing test
	fmt.Println("  - Scrolling back to top for typing test...")
	if err := stealthEngine.Scroll(page, "up", 1000); err != nil {
		fmt.Printf("⚠ Scroll to top failed: %v\n", err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println()

	// Step 13: Test typing (use the test page input)
	fmt.Println("Step 13: Testing Type() with typing simulation...")
	fmt.Println("  - Using the input field on the test page...")
	fmt.Println("  - Make sure you can see the input field on the page!")
	time.Sleep(3 * time.Second)
	
	// Use the input field from our test page
	typingSelector := "#typing-input"
	if err := page.WaitForElement(typingSelector, 10*time.Second); err != nil {
		fmt.Printf("⚠ Input field not found: %v\n", err)
		fmt.Println("  - Trying to navigate back to test page...")
		wd, _ := os.Getwd()
		testHTMLPath := fmt.Sprintf("file://%s/cmd/test-stealth/test-page.html", wd)
		if err := page.Navigate(testHTMLPath, 30*time.Second); err != nil {
			fmt.Printf("⚠ Navigation failed: %v\n", err)
			fmt.Println("  (Skipping typing test)")
		} else {
			time.Sleep(3 * time.Second)
		}
	}
	
	if err := page.WaitForElement(typingSelector, 10*time.Second); err == nil {
		fmt.Println("  - Found input field! Clicking and typing in 2 seconds...")
		fmt.Println("  - WATCH THE INPUT FIELD - text will appear character by character!")
		time.Sleep(2 * time.Second)
		
		// Click on input field first
		if err := stealthEngine.Click(page, typingSelector); err != nil {
			fmt.Printf("⚠ Failed to click input: %v\n", err)
		} else {
			time.Sleep(1 * time.Second) // Pause after click
			
			// Type text with human-like behavior
			testText := "Hello World! This is a typing test."
			fmt.Printf("  - Typing '%s' character by character...\n", testText)
			fmt.Println("  - Watch the input field fill up slowly with human-like timing!")
			if err := stealthEngine.Type(page, typingSelector, testText); err != nil {
				fmt.Printf("✗ Type failed: %v\n", err)
			} else {
				fmt.Printf("✓ Typing completed (typed: %s)\n", testText)
				fmt.Println("  - Check the input field - it should now contain the text!")
				fmt.Println("  - Check the status box for typing confirmation!")
				time.Sleep(5 * time.Second) // Longer pause to see the typed text
			}
		}
	} else {
		fmt.Printf("⚠ Input field not found: %v\n", err)
		fmt.Println("  (Skipping typing test)")
	}
	fmt.Println()

	// Step 14: Test hover (use test page button)
	fmt.Println("Step 14: Testing Hover() with mouse wander...")
	fmt.Println("  - Using button on test page...")
	fmt.Println("  - WATCH FOR THE RED CIRCLE moving to the button!")
	
	// Scroll to make sure button is visible
	fmt.Println("  - Scrolling to make button visible...")
	if err := stealthEngine.Scroll(page, "down", 200); err != nil {
		fmt.Printf("⚠ Scroll failed: %v\n", err)
	}
	time.Sleep(2 * time.Second)
	
	// Try to hover over a button
	buttonSelector := "#test-button-1"
	if err := page.WaitForElement(buttonSelector, 5*time.Second); err != nil {
		fmt.Printf("⚠ Button not found: %v\n", err)
		fmt.Println("  (This is okay - hover test skipped)")
	} else {
		fmt.Println("  - Found button! Hovering in 2 seconds...")
		fmt.Println("  - Watch the RED CIRCLE wander before settling on the button!")
		time.Sleep(2 * time.Second)
		
		if err := stealthEngine.Hover(page, buttonSelector); err != nil {
			fmt.Printf("✗ Hover failed: %v\n", err)
		} else {
			fmt.Println("✓ Hover completed (mouse wandered before settling)")
			fmt.Println("  - Check the status box on the page for hover confirmation!")
			time.Sleep(3 * time.Second) // Pause to see hover effect
		}
	}
	fmt.Println()

	// Step 15: Test click (use test page link)
	fmt.Println("Step 15: Testing Click() with human-like movement...")
	fmt.Println("  - Using link on test page...")
	fmt.Println("  - WATCH FOR THE RED CIRCLE moving in a curve before clicking!")
	
	linkSelector := "#test-link-1"
	if err := page.WaitForElement(linkSelector, 5*time.Second); err != nil {
		fmt.Printf("⚠ Link not found for click test: %v\n", err)
		fmt.Println("  (This is okay - click test skipped)")
	} else {
		fmt.Println("  - Found link! Watch the RED CIRCLE move in a curve before clicking!")
		fmt.Println("  - Starting in 2 seconds...")
		time.Sleep(2 * time.Second)
		
		if err := stealthEngine.Click(page, linkSelector); err != nil {
			fmt.Printf("✗ Click failed: %v\n", err)
		} else {
			fmt.Println("✓ Click completed (mouse moved with Bezier curve, then clicked)")
			fmt.Println("  - Check the status box on the page for click confirmation!")
			time.Sleep(3 * time.Second) // See the result
		}
	}
	fmt.Println()

	fmt.Println("=== All Stealth Tests Completed! ===")
	fmt.Println()
	fmt.Println("If you saw:")
	fmt.Println("  ✓ Curved mouse movements (not straight lines)")
	fmt.Println("  ✓ Variable speed scrolling")
	fmt.Println("  ✓ Human-like typing with pauses")
	fmt.Println("  ✓ Mouse wandering before hovering")
	fmt.Println("  ✓ Natural click behavior")
	fmt.Println()
	fmt.Println("Then Phase 7 stealth implementation is working correctly!")
	fmt.Println()
	fmt.Println("Browser will close in 5 seconds...")
	time.Sleep(5 * time.Second)
}

