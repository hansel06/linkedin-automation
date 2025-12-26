package storage

// Package storage persists state (requests, connections, messages, counters, cookies) using SQLite or similar.
//
// SQLite Schema Design:
//
// Table: profiles
//   - id TEXT PRIMARY KEY (UUID)
//   - url TEXT UNIQUE NOT NULL (LinkedIn profile URL)
//   - name TEXT (full name)
//   - title TEXT (job title)
//   - company TEXT (current company)
//   - location TEXT (location)
//   - discovered_at TIMESTAMP NOT NULL
//   - last_visited_at TIMESTAMP NOT NULL
//   - visited_count INTEGER NOT NULL DEFAULT 1
//   Indexes: idx_profiles_url (on url), idx_profiles_discovered_at (on discovered_at)
//
// Table: connection_requests
//   - id TEXT PRIMARY KEY (UUID)
//   - profile_id TEXT NOT NULL REFERENCES profiles(id)
//   - profile_url TEXT NOT NULL (denormalized for quick lookup)
//   - status TEXT NOT NULL (pending, accepted, rejected, ignored)
//   - note TEXT (personalized note)
//   - sent_at TIMESTAMP NOT NULL
//   - responded_at TIMESTAMP (nullable)
//   Indexes: idx_connection_requests_profile_id (on profile_id), idx_connection_requests_status (on status),
//            idx_connection_requests_sent_at (on sent_at), idx_connection_requests_profile_url (on profile_url)
//
// Table: messages
//   - id TEXT PRIMARY KEY (UUID)
//   - profile_id TEXT NOT NULL REFERENCES profiles(id)
//   - profile_url TEXT NOT NULL (denormalized for quick lookup)
//   - content TEXT NOT NULL (message content)
//   - status TEXT NOT NULL (sent, delivered, read, failed)
//   - sent_at TIMESTAMP NOT NULL
//   - template_id TEXT (template identifier used)
//   Indexes: idx_messages_profile_id (on profile_id), idx_messages_status (on status),
//            idx_messages_sent_at (on sent_at), idx_messages_profile_url (on profile_url)
//
// Table: daily_counters
//   - date DATE PRIMARY KEY (YYYY-MM-DD, time set to 00:00:00)
//   - connections_sent INTEGER NOT NULL DEFAULT 0
//   - messages_sent INTEGER NOT NULL DEFAULT 0
//   - last_connection_at TIMESTAMP (nullable)
//   - last_message_at TIMESTAMP (nullable)
//   Indexes: idx_daily_counters_date (on date) - already primary key
//
// Table: hourly_counters
//   - hour TIMESTAMP PRIMARY KEY (YYYY-MM-DD HH:00:00)
//   - connections_sent INTEGER NOT NULL DEFAULT 0
//   - messages_sent INTEGER NOT NULL DEFAULT 0
//   Indexes: idx_hourly_counters_hour (on hour) - already primary key
//
// Table: session_state
//   - key TEXT PRIMARY KEY (e.g., "last_run")
//   - value TEXT NOT NULL (value as string, e.g., timestamp)
//   - updated_at TIMESTAMP NOT NULL
//   Indexes: idx_session_state_key (on key) - already primary key
//
// Design decisions:
//   - Use TEXT for UUIDs (simpler than BLOB, readable in SQLite browser)
//   - Denormalize profile_url in connection_requests and messages for quick lookups without joins
//   - Use DATE type for daily_counters.date (SQLite stores as TEXT but we treat as DATE)
//   - Use TIMESTAMP for all time fields (SQLite stores as TEXT in ISO8601 format)
//   - Foreign key constraints enabled for data integrity
//   - Indexes on frequently queried columns (status, sent_at, profile_id, etc.)

import (
	"time"

	"github.com/hansel06/linkedin-automation/internal/models"
)

// Repository defines the interface for all storage operations.
// This allows us to swap implementations (SQLite, JSON, in-memory for testing) without changing calling code.
//
// All methods return errors for explicit error handling.
// Methods that query data return nil if not found (not an error - use existence checks if needed).
type Repository interface {
	// Initialize sets up the database schema (tables, indexes).
	// Should be idempotent (safe to call multiple times).
	Initialize() error

	// Close closes the database connection and cleans up resources.
	Close() error

	// Profile operations
	SaveProfile(profile *models.Profile) error
	GetProfileByURL(url string) (*models.Profile, error)
	GetProfileByID(id string) (*models.Profile, error)
	UpdateProfile(profile *models.Profile) error
	HasVisitedProfile(url string) (bool, error)
	GetAllProfiles() ([]*models.Profile, error)

	// ConnectionRequest operations
	SaveConnectionRequest(req *models.ConnectionRequest) error
	GetConnectionRequestByID(id string) (*models.ConnectionRequest, error)
	GetConnectionRequestsByProfileID(profileID string) ([]*models.ConnectionRequest, error)
	UpdateConnectionRequest(req *models.ConnectionRequest) error
	CountConnectionRequestsToday() (int, error)
	CountConnectionRequestsInHour(hour time.Time) (int, error)
	GetConnectionRequestsByStatus(status string) ([]*models.ConnectionRequest, error)
	GetPendingConnectionRequests() ([]*models.ConnectionRequest, error)
	GetAcceptedConnectionRequests() ([]*models.ConnectionRequest, error)

	// Message operations
	SaveMessage(msg *models.Message) error
	GetMessageByID(id string) (*models.Message, error)
	GetMessagesByProfileID(profileID string) ([]*models.Message, error)
	UpdateMessage(msg *models.Message) error
	CountMessagesToday() (int, error)
	CountMessagesInHour(hour time.Time) (int, error)
	GetAllMessages() ([]*models.Message, error)

	// DailyCounters operations
	GetDailyCounters(date time.Time) (*models.DailyCounters, error)
	UpdateDailyCounters(counters *models.DailyCounters) error
	IncrementConnectionCount(date time.Time) error
	IncrementMessageCount(date time.Time) error
	ResetDailyCountersIfNeeded(date time.Time) error

	// HourlyCounters operations
	GetHourlyCounters(hour time.Time) (*models.HourlyCounters, error)
	UpdateHourlyCounters(counters *models.HourlyCounters) error
	IncrementHourlyConnectionCount(hour time.Time) error
	IncrementHourlyMessageCount(hour time.Time) error

	// SessionState operations
	GetSessionState(key string) (*models.SessionState, error)
	SetSessionState(key, value string) error
	GetLastRunTime() (time.Time, error)
	SetLastRunTime(t time.Time) error
}

// SQLiteRepository is the SQLite implementation of Repository.
// The actual implementation is in sqlite.go.
// Use NewSQLiteRepository from sqlite.go to create instances.
