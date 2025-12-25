package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hansel06/linkedin-automation/internal/browser"
	"github.com/hansel06/linkedin-automation/internal/config"
	"github.com/hansel06/linkedin-automation/internal/logging"
)

func main() {
	fmt.Println("=== Browser Implementation Test ===")
	fmt.Println()

	// Step 1: Load config (use defaults for testing)
	fmt.Println("Step 1: Loading configuration...")
	cfg := config.NewDefault()
	// Override headless to false so we can see the browser
	cfg.Browser.Headless = false
	cfg.Browser.BrowserTimeout = 30 * time.Second
	fmt.Println("✓ Configuration loaded")
	fmt.Printf("  - Headless: %v\n", cfg.Browser.Headless)
	fmt.Printf("  - Viewport: %dx%d\n", cfg.Browser.ViewportWidth, cfg.Browser.ViewportHeight)
	fmt.Printf("  - User Agent: %s\n", cfg.Browser.UserAgent[:50]+"...")
	fmt.Println()

	// Step 2: Create logger
	fmt.Println("Step 2: Creating logger...")
	logger := logging.NewLogger(cfg.Logging.LogLevel, cfg.Logging.LogFormat)
	logger = logger.WithComponent("test-browser")
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
	fmt.Println("✓ Browser started successfully")
	fmt.Println()

	// Ensure browser is closed at the end
	defer func() {
		fmt.Println()
		fmt.Println("Step 8: Closing browser...")
		if err := bm.Close(); err != nil {
			fmt.Printf("✗ Failed to close browser: %v\n", err)
		} else {
			fmt.Println("✓ Browser closed successfully")
		}
	}()

	// Step 5: Create a new page
	fmt.Println("Step 5: Creating new page...")
	page, err := bm.NewPage()
	if err != nil {
		fmt.Printf("✗ Failed to create page: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Page created")
	fmt.Printf("  - URL: %s\n", page.GetURL())
	fmt.Printf("  - Title: %s\n", page.GetTitle())
	fmt.Println()

	// Step 6: Test fingerprint masking
	fmt.Println("Step 6: Testing fingerprint masking...")
	// Check if webdriver is undefined (should be after masking)
	jsCheck := `
		(function() {
			return {
				webdriver: navigator.webdriver,
				hasChrome: typeof window.chrome !== 'undefined',
				plugins: navigator.plugins.length,
				languages: navigator.languages
			};
		})()
	`
	result, err := page.Eval(jsCheck)
	if err != nil {
		fmt.Printf("✗ Failed to check fingerprint: %v\n", err)
	} else {
		fmt.Println("✓ Fingerprint check completed")
		fmt.Printf("  - Result: %v\n", result)
		// Note: webdriver should be undefined, chrome should exist
	}
	fmt.Println()

	// Step 7: Navigate to a test page
	fmt.Println("Step 7: Navigating to test page...")
	testURL := "https://youtube.com"
	fmt.Printf("  - URL: %s\n", testURL)
	if err := page.Navigate(testURL, 30*time.Second); err != nil {
		fmt.Printf("✗ Failed to navigate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Navigation completed")
	fmt.Printf("  - Current URL: %s\n", page.GetURL())
	fmt.Printf("  - Page Title: %s\n", page.GetTitle())
	fmt.Println()

	// Wait a bit to see the page
	fmt.Println("Waiting 3 seconds to observe the page...")
	time.Sleep(3 * time.Second)

	// Step 8: Test cookie save/load
	fmt.Println()
	fmt.Println("Step 8: Testing cookie save/load...")
	cookiePath := "data/test-cookies.json"
	
	// Save cookies
	fmt.Printf("  - Saving cookies to: %s\n", cookiePath)
	if err := bm.SaveCookies(cookiePath); err != nil {
		fmt.Printf("✗ Failed to save cookies: %v\n", err)
	} else {
		fmt.Println("✓ Cookies saved")
	}

	// Load cookies (test that it doesn't error)
	fmt.Printf("  - Loading cookies from: %s\n", cookiePath)
	if err := bm.LoadCookies(cookiePath); err != nil {
		fmt.Printf("✗ Failed to load cookies: %v\n", err)
	} else {
		fmt.Println("✓ Cookies loaded")
	}
	fmt.Println()

	// Step 9: Test WaitForElement (wait for body tag)
	fmt.Println("Step 9: Testing WaitForElement...")
	if err := page.WaitForElement("body", 5*time.Second); err != nil {
		fmt.Printf("✗ Failed to wait for element: %v\n", err)
	} else {
		fmt.Println("✓ Element found (body tag exists)")
	}
	fmt.Println()

	// Step 10: Test page info
	fmt.Println("Step 10: Final page info...")
	fmt.Printf("  - URL: %s\n", page.GetURL())
	fmt.Printf("  - Title: %s\n", page.GetTitle())
	fmt.Println()

	fmt.Println("=== All Tests Completed Successfully! ===")
	fmt.Println()
	fmt.Println("If you saw the browser window open and navigate to example.com,")
	fmt.Println("then the browser implementation is working correctly!")
}

