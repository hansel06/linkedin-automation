package models

// Package models defines shared data structures (profiles, messages, counters, session metadata).

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a LinkedIn profile that we've discovered or interacted with.
type Profile struct {
	ID          string    // Unique identifier (UUID)
	URL         string    // LinkedIn profile URL (e.g., "https://linkedin.com/in/username")
	Name        string    // Full name (if available)
	Title       string    // Job title (if available)
	Company     string    // Current company (if available)
	Location    string    // Location (if available)
	DiscoveredAt time.Time // When we first found this profile
	LastVisitedAt time.Time // When we last visited this profile
	VisitedCount int      // How many times we've visited
}

// ConnectionRequest represents a connection request we've sent.
type ConnectionRequest struct {
	ID          string    // Unique identifier (UUID)
	ProfileID   string    // Reference to Profile.ID
	ProfileURL  string    // Profile URL (for quick lookup)
	Status      string    // "pending", "accepted", "rejected", "ignored"
	Note        string    // Personalized note sent with request (if any)
	SentAt      time.Time // When we sent the request
	RespondedAt *time.Time // When they responded (nil if pending)
}

// Message represents a message we've sent to a connection.
type Message struct {
	ID          string    // Unique identifier (UUID)
	ProfileID   string    // Reference to Profile.ID
	ProfileURL  string    // Profile URL (for quick lookup)
	Content     string    // Message content
	Status      string    // "sent", "delivered", "read", "failed"
	SentAt      time.Time // When we sent the message
	TemplateID  string    // Template identifier used (if any)
}

// DailyCounters tracks daily and hourly limits for rate limiting.
type DailyCounters struct {
	Date              time.Time // Date (YYYY-MM-DD, time set to 00:00:00)
	ConnectionsSent  int       // Number of connection requests sent today
	MessagesSent     int       // Number of messages sent today
	LastConnectionAt time.Time // Timestamp of last connection request
	LastMessageAt    time.Time // Timestamp of last message
}

// HourlyCounters tracks hourly limits (subset of daily counters).
// This is embedded in DailyCounters for simplicity, but we track hourly separately.
type HourlyCounters struct {
	Hour              time.Time // Hour (YYYY-MM-DD HH:00:00)
	ConnectionsSent   int       // Number of connection requests sent this hour
	MessagesSent      int       // Number of messages sent this hour
}

// SessionState tracks application session metadata.
type SessionState struct {
	Key         string    // Unique key (e.g., "last_run")
	Value       string    // Value (e.g., timestamp as string)
	UpdatedAt   time.Time // When this was last updated
}

// NewProfile creates a new Profile with generated ID and timestamps.
func NewProfile(url string) *Profile {
	now := time.Now()
	return &Profile{
		ID:           uuid.New().String(),
		URL:          url,
		Name:         "",
		Title:        "",
		Company:      "",
		Location:     "",
		DiscoveredAt: now,
		LastVisitedAt: now,
		VisitedCount: 1,
	}
}

// NewConnectionRequest creates a new ConnectionRequest with generated ID and timestamp.
func NewConnectionRequest(profileID, profileURL, note string) *ConnectionRequest {
	return &ConnectionRequest{
		ID:         uuid.New().String(),
		ProfileID:  profileID,
		ProfileURL: profileURL,
		Status:     "pending",
		Note:       note,
		SentAt:     time.Now(),
		RespondedAt: nil,
	}
}

// NewMessage creates a new Message with generated ID and timestamp.
func NewMessage(profileID, profileURL, content, templateID string) *Message {
	return &Message{
		ID:         uuid.New().String(),
		ProfileID:  profileID,
		ProfileURL: profileURL,
		Content:    content,
		Status:     "sent",
		SentAt:     time.Now(),
		TemplateID: templateID,
	}
}
