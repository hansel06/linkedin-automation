package config

// Package config loads and validates configuration from files and environment overrides.

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
// It combines settings from defaults, YAML file, and environment variables.
type Config struct {
	Credentials Credentials `yaml:"-"` // Never in YAML, env-only
	Browser     BrowserConfig
	Stealth     StealthConfig
	Limits      LimitsConfig
	Schedule    ScheduleConfig
	Storage     StorageConfig
	Logging     LoggingConfig
	Search      SearchConfig
}

// Credentials holds sensitive authentication data (env-only).
type Credentials struct {
	Email    string
	Password string
}

// BrowserConfig holds browser-related settings.
type BrowserConfig struct {
	Headless       bool          `yaml:"headless"`
	BrowserTimeout time.Duration `yaml:"browser_timeout"`
	ViewportWidth  int           `yaml:"viewport_width"`
	ViewportHeight int           `yaml:"viewport_height"`
	UserAgent      string        `yaml:"user_agent"`
}

// StealthConfig holds anti-detection timing and behavior settings.
type StealthConfig struct {
	MinDelay          time.Duration `yaml:"min_delay"`
	MaxDelay          time.Duration `yaml:"max_delay"`
	ScrollSpeedMin    int           `yaml:"scroll_speed_min"`
	ScrollSpeedMax    int           `yaml:"scroll_speed_max"`
	TypingSpeedMin    time.Duration `yaml:"typing_speed_min"`
	TypingSpeedMax    time.Duration `yaml:"typing_speed_max"`
	MouseMovementSpeed float64      `yaml:"mouse_movement_speed"`
}

// LimitsConfig holds rate limiting settings.
type LimitsConfig struct {
	DailyConnectLimit  int           `yaml:"daily_connect_limit"`
	HourlyConnectLimit int           `yaml:"hourly_connect_limit"`
	MessageCooldown    time.Duration `yaml:"message_cooldown"`
	ConnectCooldown    time.Duration `yaml:"connect_cooldown"`
}

// ScheduleConfig holds activity scheduling settings.
type ScheduleConfig struct {
	BusinessHoursStart string `yaml:"business_hours_start"` // Format: "HH:MM"
	BusinessHoursEnd   string `yaml:"business_hours_end"`   // Format: "HH:MM"
	Timezone           string `yaml:"timezone"`             // e.g., "America/New_York"
}

// StorageConfig holds file path settings for persistence.
type StorageConfig struct {
	DatabasePath string `yaml:"database_path"`
	CookiePath   string `yaml:"cookie_path"`
}

// LoggingConfig holds logging output settings.
type LoggingConfig struct {
	LogLevel  string `yaml:"log_level"`  // "debug", "info", "warn", "error"
	LogFormat string `yaml:"log_format"` // "json" or "console"
}

// SearchConfig holds search pagination settings.
type SearchConfig struct {
	MaxSearchPages int `yaml:"max_search_pages"`
	ResultsPerPage int `yaml:"results_per_page"`
}

// Default values for all config fields.
var (
	// Browser defaults
	defaultHeadless       = true
	defaultBrowserTimeout = 30 * time.Second
	defaultViewportWidth  = 1920
	defaultViewportHeight = 1080
	defaultUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Stealth defaults
	defaultMinDelay          = 2 * time.Second
	defaultMaxDelay          = 5 * time.Second
	defaultScrollSpeedMin    = 100
	defaultScrollSpeedMax    = 300
	defaultTypingSpeedMin    = 50 * time.Millisecond
	defaultTypingSpeedMax    = 150 * time.Millisecond
	defaultMouseMovementSpeed = 0.8

	// Limits defaults
	defaultDailyConnectLimit  = 50
	defaultHourlyConnectLimit = 10
	defaultMessageCooldown    = 30 * time.Second
	defaultConnectCooldown    = 10 * time.Second

	// Schedule defaults
	defaultBusinessHoursStart = "09:00"
	defaultBusinessHoursEnd   = "17:00"
	defaultTimezone           = "America/New_York"

	// Storage defaults
	defaultDatabasePath = "data/automation.db"
	defaultCookiePath   = "data/cookies.json"

	// Logging defaults
	defaultLogLevel  = "info"
	defaultLogFormat = "console"

	// Search defaults
	defaultMaxSearchPages = 10
	defaultResultsPerPage = 10
)

// NewDefault returns a Config with all default values set.
func NewDefault() *Config {
	return &Config{
		Credentials: Credentials{
			Email:    "",
			Password: "",
		},
		Browser: BrowserConfig{
			Headless:       defaultHeadless,
			BrowserTimeout: defaultBrowserTimeout,
			ViewportWidth:  defaultViewportWidth,
			ViewportHeight: defaultViewportHeight,
			UserAgent:      defaultUserAgent,
		},
		Stealth: StealthConfig{
			MinDelay:          defaultMinDelay,
			MaxDelay:          defaultMaxDelay,
			ScrollSpeedMin:    defaultScrollSpeedMin,
			ScrollSpeedMax:    defaultScrollSpeedMax,
			TypingSpeedMin:    defaultTypingSpeedMin,
			TypingSpeedMax:    defaultTypingSpeedMax,
			MouseMovementSpeed: defaultMouseMovementSpeed,
		},
		Limits: LimitsConfig{
			DailyConnectLimit:  defaultDailyConnectLimit,
			HourlyConnectLimit: defaultHourlyConnectLimit,
			MessageCooldown:    defaultMessageCooldown,
			ConnectCooldown:    defaultConnectCooldown,
		},
		Schedule: ScheduleConfig{
			BusinessHoursStart: defaultBusinessHoursStart,
			BusinessHoursEnd:   defaultBusinessHoursEnd,
			Timezone:           defaultTimezone,
		},
		Storage: StorageConfig{
			DatabasePath: defaultDatabasePath,
			CookiePath:   defaultCookiePath,
		},
		Logging: LoggingConfig{
			LogLevel:  defaultLogLevel,
			LogFormat: defaultLogFormat,
		},
		Search: SearchConfig{
			MaxSearchPages: defaultMaxSearchPages,
			ResultsPerPage: defaultResultsPerPage,
		},
	}
}

// Load loads configuration from multiple sources in order of precedence:
// 1. Default values (hardcoded)
// 2. YAML config file (if exists, overrides defaults)
// 3. Environment variables (override YAML if present)
//
// Config file path is determined by CONFIG_PATH env var, or defaults to "config.yaml".
func Load() (*Config, error) {
	// Start with defaults
	cfg := NewDefault()

	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()

	// Determine config file path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Load YAML config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}

		var yamlConfig Config
		if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}

		// Merge YAML config into defaults (YAML overrides defaults)
		cfg.mergeYAML(&yamlConfig)
	}

	// Apply environment variable overrides (env overrides YAML)
	cfg.applyEnvOverrides()

	// Validate the final config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// mergeYAML merges YAML config values into the current config.
// Only non-zero values from YAML override defaults.
func (c *Config) mergeYAML(yaml *Config) {
	// Browser
	if yaml.Browser.Headless != false || yaml.Browser.Headless == true {
		c.Browser.Headless = yaml.Browser.Headless
	}
	if yaml.Browser.BrowserTimeout != 0 {
		c.Browser.BrowserTimeout = yaml.Browser.BrowserTimeout
	}
	if yaml.Browser.ViewportWidth != 0 {
		c.Browser.ViewportWidth = yaml.Browser.ViewportWidth
	}
	if yaml.Browser.ViewportHeight != 0 {
		c.Browser.ViewportHeight = yaml.Browser.ViewportHeight
	}
	if yaml.Browser.UserAgent != "" {
		c.Browser.UserAgent = yaml.Browser.UserAgent
	}

	// Stealth
	if yaml.Stealth.MinDelay != 0 {
		c.Stealth.MinDelay = yaml.Stealth.MinDelay
	}
	if yaml.Stealth.MaxDelay != 0 {
		c.Stealth.MaxDelay = yaml.Stealth.MaxDelay
	}
	if yaml.Stealth.ScrollSpeedMin != 0 {
		c.Stealth.ScrollSpeedMin = yaml.Stealth.ScrollSpeedMin
	}
	if yaml.Stealth.ScrollSpeedMax != 0 {
		c.Stealth.ScrollSpeedMax = yaml.Stealth.ScrollSpeedMax
	}
	if yaml.Stealth.TypingSpeedMin != 0 {
		c.Stealth.TypingSpeedMin = yaml.Stealth.TypingSpeedMin
	}
	if yaml.Stealth.TypingSpeedMax != 0 {
		c.Stealth.TypingSpeedMax = yaml.Stealth.TypingSpeedMax
	}
	if yaml.Stealth.MouseMovementSpeed != 0 {
		c.Stealth.MouseMovementSpeed = yaml.Stealth.MouseMovementSpeed
	}

	// Limits
	if yaml.Limits.DailyConnectLimit != 0 {
		c.Limits.DailyConnectLimit = yaml.Limits.DailyConnectLimit
	}
	if yaml.Limits.HourlyConnectLimit != 0 {
		c.Limits.HourlyConnectLimit = yaml.Limits.HourlyConnectLimit
	}
	if yaml.Limits.MessageCooldown != 0 {
		c.Limits.MessageCooldown = yaml.Limits.MessageCooldown
	}
	if yaml.Limits.ConnectCooldown != 0 {
		c.Limits.ConnectCooldown = yaml.Limits.ConnectCooldown
	}

	// Schedule
	if yaml.Schedule.BusinessHoursStart != "" {
		c.Schedule.BusinessHoursStart = yaml.Schedule.BusinessHoursStart
	}
	if yaml.Schedule.BusinessHoursEnd != "" {
		c.Schedule.BusinessHoursEnd = yaml.Schedule.BusinessHoursEnd
	}
	if yaml.Schedule.Timezone != "" {
		c.Schedule.Timezone = yaml.Schedule.Timezone
	}

	// Storage
	if yaml.Storage.DatabasePath != "" {
		c.Storage.DatabasePath = yaml.Storage.DatabasePath
	}
	if yaml.Storage.CookiePath != "" {
		c.Storage.CookiePath = yaml.Storage.CookiePath
	}

	// Logging
	if yaml.Logging.LogLevel != "" {
		c.Logging.LogLevel = yaml.Logging.LogLevel
	}
	if yaml.Logging.LogFormat != "" {
		c.Logging.LogFormat = yaml.Logging.LogFormat
	}

	// Search
	if yaml.Search.MaxSearchPages != 0 {
		c.Search.MaxSearchPages = yaml.Search.MaxSearchPages
	}
	if yaml.Search.ResultsPerPage != 0 {
		c.Search.ResultsPerPage = yaml.Search.ResultsPerPage
	}
}

// applyEnvOverrides applies environment variable overrides to config.
// Environment variables take highest precedence.
func (c *Config) applyEnvOverrides() {
	// Credentials (env-only)
	if email := os.Getenv("EMAIL"); email != "" {
		c.Credentials.Email = email
	}
	if password := os.Getenv("PASSWORD"); password != "" {
		c.Credentials.Password = password
	}

	// Browser settings (only bool/string/int overrides, durations skipped for simplicity)
	// Note: For duration/int overrides, user should use YAML file

	// Logging (commonly overridden via env)
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.LogLevel = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		c.Logging.LogFormat = format
	}

	// Storage paths
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		c.Storage.DatabasePath = dbPath
	}
	if cookiePath := os.Getenv("COOKIE_PATH"); cookiePath != "" {
		c.Storage.CookiePath = cookiePath
	}
}

// Validate validates all configuration values and returns an error if invalid.
func (c *Config) Validate() error {
	// Required fields
	if c.Credentials.Email == "" {
		return fmt.Errorf("email is required (set EMAIL environment variable)")
	}
	if c.Credentials.Password == "" {
		return fmt.Errorf("password is required (set PASSWORD environment variable)")
	}

	// Browser validation
	if c.Browser.ViewportWidth < 800 || c.Browser.ViewportWidth > 3840 {
		return fmt.Errorf("viewport_width must be between 800 and 3840, got %d", c.Browser.ViewportWidth)
	}
	if c.Browser.ViewportHeight < 600 || c.Browser.ViewportHeight > 2160 {
		return fmt.Errorf("viewport_height must be between 600 and 2160, got %d", c.Browser.ViewportHeight)
	}
	if c.Browser.UserAgent == "" {
		return fmt.Errorf("user_agent cannot be empty")
	}

	// Stealth validation
	if c.Stealth.MinDelay >= c.Stealth.MaxDelay {
		return fmt.Errorf("min_delay (%v) must be less than max_delay (%v)", c.Stealth.MinDelay, c.Stealth.MaxDelay)
	}
	if c.Stealth.MinDelay < 0 {
		return fmt.Errorf("min_delay cannot be negative")
	}
	if c.Stealth.ScrollSpeedMin > c.Stealth.ScrollSpeedMax {
		return fmt.Errorf("scroll_speed_min (%d) must be less than or equal to scroll_speed_max (%d)", c.Stealth.ScrollSpeedMin, c.Stealth.ScrollSpeedMax)
	}
	if c.Stealth.ScrollSpeedMin < 0 {
		return fmt.Errorf("scroll_speed_min cannot be negative")
	}
	if c.Stealth.TypingSpeedMin > c.Stealth.TypingSpeedMax {
		return fmt.Errorf("typing_speed_min (%v) must be less than or equal to typing_speed_max (%v)", c.Stealth.TypingSpeedMin, c.Stealth.TypingSpeedMax)
	}
	if c.Stealth.TypingSpeedMin < 0 {
		return fmt.Errorf("typing_speed_min cannot be negative")
	}
	if c.Stealth.MouseMovementSpeed <= 0 {
		return fmt.Errorf("mouse_movement_speed must be positive, got %f", c.Stealth.MouseMovementSpeed)
	}

	// Limits validation
	if c.Limits.DailyConnectLimit <= 0 {
		return fmt.Errorf("daily_connect_limit must be positive, got %d", c.Limits.DailyConnectLimit)
	}
	if c.Limits.HourlyConnectLimit <= 0 {
		return fmt.Errorf("hourly_connect_limit must be positive, got %d", c.Limits.HourlyConnectLimit)
	}
	if c.Limits.HourlyConnectLimit > c.Limits.DailyConnectLimit {
		return fmt.Errorf("hourly_connect_limit (%d) cannot exceed daily_connect_limit (%d)", c.Limits.HourlyConnectLimit, c.Limits.DailyConnectLimit)
	}
	if c.Limits.MessageCooldown < 0 {
		return fmt.Errorf("message_cooldown cannot be negative")
	}
	if c.Limits.ConnectCooldown < 0 {
		return fmt.Errorf("connect_cooldown cannot be negative")
	}

	// Schedule validation
	if err := validateTimeFormat(c.Schedule.BusinessHoursStart); err != nil {
		return fmt.Errorf("business_hours_start: %w", err)
	}
	if err := validateTimeFormat(c.Schedule.BusinessHoursEnd); err != nil {
		return fmt.Errorf("business_hours_end: %w", err)
	}
	startTime, err := parseTime(c.Schedule.BusinessHoursStart)
	if err != nil {
		return fmt.Errorf("business_hours_start: %w", err)
	}
	endTime, err := parseTime(c.Schedule.BusinessHoursEnd)
	if err != nil {
		return fmt.Errorf("business_hours_end: %w", err)
	}
	if startTime.After(endTime) || startTime.Equal(endTime) {
		return fmt.Errorf("business_hours_start (%s) must be before business_hours_end (%s)", c.Schedule.BusinessHoursStart, c.Schedule.BusinessHoursEnd)
	}
	if c.Schedule.Timezone != "" {
		if _, err := time.LoadLocation(c.Schedule.Timezone); err != nil {
			return fmt.Errorf("invalid timezone %s: %w", c.Schedule.Timezone, err)
		}
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error, got %s", c.Logging.LogLevel)
	}
	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[c.Logging.LogFormat] {
		return fmt.Errorf("log_format must be one of: json, console, got %s", c.Logging.LogFormat)
	}

	// Search validation
	if c.Search.MaxSearchPages <= 0 {
		return fmt.Errorf("max_search_pages must be positive, got %d", c.Search.MaxSearchPages)
	}
	if c.Search.ResultsPerPage <= 0 {
		return fmt.Errorf("results_per_page must be positive, got %d", c.Search.ResultsPerPage)
	}

	return nil
}

// validateTimeFormat validates that a time string is in "HH:MM" format.
func validateTimeFormat(timeStr string) error {
	if len(timeStr) != 5 {
		return fmt.Errorf("time must be in HH:MM format, got %s", timeStr)
	}
	if timeStr[2] != ':' {
		return fmt.Errorf("time must be in HH:MM format, got %s", timeStr)
	}
	return nil
}

// parseTime parses a "HH:MM" time string.
func parseTime(timeStr string) (time.Time, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %w", err)
	}
	return t, nil
}
