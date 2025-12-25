package browser

// Package browser wraps Rod setup, fingerprint masking, and page/session management.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/hansel06/linkedin-automation/internal/config"
	"github.com/hansel06/linkedin-automation/internal/logging"
)

// BrowserManager manages browser lifecycle, pages, fingerprint masking, and cookies.
// It provides a low-level wrapper around Rod for browser automation.
// All browser interactions should go through the stealth layer, not directly through BrowserManager.
type BrowserManager struct {
	browser   *rod.Browser // Rod browser instance
	config    *config.BrowserConfig
	logger    *logging.Logger
	activePage *Page      // Currently active page (one at a time)
	pages     []*Page     // All open pages (supports multiple pages)
}

// Page wraps a Rod page and provides safe access to page operations.
// It does not expose Rod's page directly to prevent bypassing the stealth layer.
type Page struct {
	page    *rod.Page // Internal Rod page (not exported)
	browser *BrowserManager
	url     string
	title   string
}

// NewBrowserManager creates a new browser manager instance.
// It does not launch the browser - call Start() to launch.
func NewBrowserManager(cfg *config.BrowserConfig, logger *logging.Logger) (*BrowserManager, error) {
	if cfg == nil {
		return nil, errors.New("browser config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	return &BrowserManager{
		config:     cfg,
		logger:     logger,
		activePage: nil,
		pages:      make([]*Page, 0),
	}, nil
}

// Start launches the browser instance with configured settings.
// It applies headless mode, viewport size, and user agent from config.
func (bm *BrowserManager) Start() error {
	if bm.browser != nil {
		return errors.New("browser is already started")
	}

	bm.logger.Info().Msg("Starting browser")

	// Create launcher with headless mode
	l := launcher.New()
	if bm.config.Headless {
		l = l.Headless(true)
	} else {
		l = l.Headless(false)
	}

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	bm.browser = browser
	bm.logger.Info().
		Bool("headless", bm.config.Headless).
		Msg("Browser started successfully")

	return nil
}

// Close shuts down the browser and all associated pages.
// It should be called when the browser is no longer needed.
func (bm *BrowserManager) Close() error {
	if bm.browser == nil {
		return nil // Already closed, not an error
	}

	bm.logger.Info().Msg("Closing browser")

	// Close all pages first
	for _, page := range bm.pages {
		if err := page.page.Close(); err != nil {
			bm.logger.Warn().Err(err).Msg("Error closing page during browser shutdown")
		}
	}

	// Close browser
	if err := bm.browser.Close(); err != nil {
		return fmt.Errorf("failed to close browser: %w", err)
	}

	// Clear internal state
	bm.browser = nil
	bm.activePage = nil
	bm.pages = make([]*Page, 0)

	bm.logger.Info().Msg("Browser closed successfully")
	return nil
}

// NewPage creates a new page and applies fingerprint masking.
// The new page becomes the active page.
// Fingerprint masking includes: disabling navigator.webdriver, setting UA, viewport, etc.
func (bm *BrowserManager) NewPage() (*Page, error) {
	if bm.browser == nil {
		return nil, errors.New("browser not started - call Start() first")
	}

	bm.logger.Info().Msg("Creating new page")

	// Create new page
	rodPage := bm.browser.MustPage("")

	// Set viewport size
	rodPage.MustSetViewport(bm.config.ViewportWidth, bm.config.ViewportHeight, 1, false)

	// Set user agent via JavaScript (Rod's SetUserAgent requires proto struct, so we use JS)
	if bm.config.UserAgent != "" {
		uaJS := fmt.Sprintf(`Object.defineProperty(navigator, 'userAgent', { get: () => '%s' });`, bm.config.UserAgent)
		rodPage.MustEvalOnNewDocument(uaJS)
	}

	// Create our Page wrapper
	page := &Page{
		page:    rodPage,
		browser: bm,
		url:     "",
		title:   "",
	}

	// Apply fingerprint masking (before any navigation)
	if err := bm.MaskFingerprint(page); err != nil {
		rodPage.Close()
		return nil, fmt.Errorf("failed to apply fingerprint masking: %w", err)
	}

	// Add to pages slice
	bm.pages = append(bm.pages, page)

	// Set as active page
	bm.activePage = page

	bm.logger.Info().Msg("Page created and fingerprint masking applied")
	return page, nil
}

// GetActivePage returns the currently active page.
// Returns an error if no page is active.
func (bm *BrowserManager) GetActivePage() (*Page, error) {
	if bm.activePage == nil {
		return nil, errors.New("no active page")
	}
	return bm.activePage, nil
}

// SetActivePage sets which page is currently active.
// The page must be one of the pages managed by this browser manager.
func (bm *BrowserManager) SetActivePage(page *Page) error {
	if page == nil {
		return errors.New("page cannot be nil")
	}

	// Verify page is in our pages slice
	found := false
	for _, p := range bm.pages {
		if p == page {
			found = true
			break
		}
	}

	if !found {
		return errors.New("page is not managed by this browser manager")
	}

	bm.activePage = page
	bm.logger.Debug().Str("url", page.url).Msg("Active page changed")
	return nil
}

// MaskFingerprint applies anti-detection techniques to a page.
// This includes:
// - Disabling navigator.webdriver flag
// - Setting realistic browser properties (plugins, permissions)
// - Ensuring window.chrome exists
// - Randomizing browser properties
func (bm *BrowserManager) MaskFingerprint(page *Page) error {
	if page == nil || page.page == nil {
		return errors.New("page cannot be nil")
	}

	// JavaScript to mask fingerprint - inject before any page loads
	js := `
		// Remove webdriver flag
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});

		// Add window.chrome object (Chrome browser indicator)
		window.chrome = {
			runtime: {}
		};

		// Set navigator.plugins to realistic values
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5] // Array with length (simulates plugins)
		});

		// Set navigator.languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Ensure navigator.permissions exists
		if (!navigator.permissions) {
			navigator.permissions = {
				query: () => Promise.resolve({ state: 'granted' })
			};
		}
	`

	// Inject JavaScript that runs on every new document (before page loads)
	_, err := page.page.EvalOnNewDocument(js)
	if err != nil {
		return fmt.Errorf("failed to inject fingerprint masking: %w", err)
	}

	bm.logger.Debug().Str("url", page.url).Msg("Fingerprint masking applied")
	return nil
}

// SaveCookies saves all cookies from all pages to a file.
// The file path is specified in the storage config.
func (bm *BrowserManager) SaveCookies(path string) error {
	if bm.browser == nil {
		return errors.New("browser not started")
	}

	if bm.activePage == nil || bm.activePage.page == nil {
		bm.logger.Warn().Msg("No active page to save cookies from")
		return nil // Not an error - just no cookies to save
	}

	bm.logger.Info().Str("path", path).Msg("Saving cookies")

	// Get cookies from active page
	cookies, err := bm.activePage.page.Cookies([]string{})
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for cookies: %w", err)
	}

	// Serialize cookies to JSON
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	bm.logger.Info().
		Str("path", path).
		Int("count", len(cookies)).
		Msg("Cookies saved successfully")

	return nil
}

// LoadCookies loads cookies from a file and applies them to the browser.
// This allows session persistence across runs.
func (bm *BrowserManager) LoadCookies(path string) error {
	if bm.browser == nil {
		return errors.New("browser not started")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		bm.logger.Debug().Str("path", path).Msg("Cookie file does not exist (first run)")
		return nil // Not an error - first run, no cookies to load
	}

	bm.logger.Info().Str("path", path).Msg("Loading cookies")

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read cookies file: %w", err)
	}

	// Deserialize cookies from JSON (as NetworkCookie, then convert to NetworkCookieParam)
	var networkCookies []*proto.NetworkCookie
	if err := json.Unmarshal(data, &networkCookies); err != nil {
		return fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	// Convert NetworkCookie to NetworkCookieParam for SetCookies
	cookieParams := make([]*proto.NetworkCookieParam, len(networkCookies))
	for i, cookie := range networkCookies {
		cookieParams[i] = &proto.NetworkCookieParam{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: cookie.SameSite,
			Expires:  cookie.Expires,
		}
	}

	// Apply cookies to active page (or create one if needed)
	if bm.activePage == nil || bm.activePage.page == nil {
		// Create a page to apply cookies to
		page, err := bm.NewPage()
		if err != nil {
			return fmt.Errorf("failed to create page for loading cookies: %w", err)
		}
		bm.activePage = page
	}

	// Set cookies
	if err := bm.activePage.page.SetCookies(cookieParams); err != nil {
		return fmt.Errorf("failed to set cookies: %w", err)
	}

	bm.logger.Info().
		Str("path", path).
		Int("count", len(cookieParams)).
		Msg("Cookies loaded successfully")

	return nil
}

// Navigate navigates the page to the specified URL with a timeout.
// It waits for the page to load completely before returning.
// Uses the browser timeout from config, or the provided timeout if smaller.
func (p *Page) Navigate(url string, timeout time.Duration) error {
	if p.page == nil {
		return errors.New("page is closed")
	}

	// Use the smaller of the two timeouts
	actualTimeout := timeout
	if timeout > p.browser.config.BrowserTimeout {
		actualTimeout = p.browser.config.BrowserTimeout
	}

	p.browser.logger.Info().Str("url", url).Dur("timeout", actualTimeout).Msg("Navigating to URL")

	// Set timeout for this operation
	p.page = p.page.Timeout(actualTimeout)

	// Navigate and wait for load
	if err := p.page.Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Wait for page to be idle (no pending network requests)
	if err := p.page.WaitIdle(time.Second); err != nil {
		return fmt.Errorf("page did not become idle after navigation: %w", err)
	}

	// Update cached URL and title
	p.url = p.page.MustInfo().URL
	p.title = p.page.MustInfo().Title

	p.browser.logger.Info().
		Str("url", p.url).
		Str("title", p.title).
		Msg("Navigation completed")

	return nil
}

// WaitForElement waits for an element matching the selector to appear on the page.
// Returns an error if the element does not appear within the timeout.
func (p *Page) WaitForElement(selector string, timeout time.Duration) error {
	if p.page == nil {
		return errors.New("page is closed")
	}

	p.browser.logger.Debug().Str("selector", selector).Dur("timeout", timeout).Msg("Waiting for element")

	// Set timeout and wait for element
	p.page = p.page.Timeout(timeout)
	_, err := p.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element %s not found within timeout: %w", selector, err)
	}

	p.browser.logger.Debug().Str("selector", selector).Msg("Element found")
	return nil
}

// WaitForNavigation waits for page navigation to complete.
// This is useful after clicking links or submitting forms.
func (p *Page) WaitForNavigation(timeout time.Duration) error {
	if p.page == nil {
		return errors.New("page is closed")
	}

	p.browser.logger.Debug().Dur("timeout", timeout).Msg("Waiting for navigation")

	// Set timeout and wait for page to load
	p.page = p.page.Timeout(timeout)
	p.page.WaitLoad()

	// Wait for page to be idle
	if err := p.page.WaitIdle(time.Second); err != nil {
		return fmt.Errorf("page did not become idle after navigation: %w", err)
	}

	// Update cached URL and title
	p.url = p.page.MustInfo().URL
	p.title = p.page.MustInfo().Title

	p.browser.logger.Debug().
		Str("url", p.url).
		Msg("Navigation completed")

	return nil
}

// Eval executes JavaScript code on the page and returns the result.
// The JavaScript code should return a value (not just execute side effects).
func (p *Page) Eval(js string) (interface{}, error) {
	if p.page == nil {
		return nil, errors.New("page is closed")
	}

	p.browser.logger.Debug().Str("js", js).Msg("Executing JavaScript")
	result, err := p.page.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("JavaScript execution failed: %w", err)
	}

	return result.Value, nil
}

// GetURL returns the current page URL.
func (p *Page) GetURL() string {
	return p.url
}

// GetTitle returns the current page title.
func (p *Page) GetTitle() string {
	return p.title
}

// RodPage returns the underlying Rod page for low-level operations.
// This is used by the stealth layer for mouse and keyboard operations.
// Should not be used directly by feature code - use stealth layer instead.
func (p *Page) RodPage() *rod.Page {
	return p.page
}

// Close closes this page.
// If this was the active page, the active page is set to nil.
func (p *Page) Close() error {
	if p.page == nil {
		return nil // Already closed
	}

	p.browser.logger.Info().Str("url", p.url).Msg("Closing page")

	// Close the Rod page
	if err := p.page.Close(); err != nil {
		return fmt.Errorf("failed to close page: %w", err)
	}

	// Remove from pages slice
	newPages := make([]*Page, 0)
	for _, page := range p.browser.pages {
		if page != p {
			newPages = append(newPages, page)
		}
	}
	p.browser.pages = newPages

	// Clear active page if this was it
	if p.browser.activePage == p {
		p.browser.activePage = nil
	}

	// Mark page as closed
	p.page = nil

	p.browser.logger.Info().Msg("Page closed")
	return nil
}
