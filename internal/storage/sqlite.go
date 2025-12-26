package storage

// Package storage implements SQLite persistence for the Repository interface.

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
	"github.com/hansel06/linkedin-automation/internal/models"
	"github.com/hansel06/linkedin-automation/internal/logging"
)

// SQLiteRepository is the SQLite implementation of Repository.
type SQLiteRepository struct {
	db     *sql.DB
	dbPath string
	logger *logging.Logger
}

// NewSQLiteRepository creates a new SQLite repository instance.
// It opens a connection to the SQLite database at the specified path.
// The database file and directory will be created if they don't exist.
func NewSQLiteRepository(dbPath string, logger *logging.Logger) (*SQLiteRepository, error) {
	// Create data directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	// Open SQLite database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &SQLiteRepository{
		db:     db,
		dbPath: dbPath,
		logger: logger,
	}

	logger.Info().Str("path", dbPath).Msg("SQLite repository initialized")

	return repo, nil
}

// Close closes the database connection and cleans up resources.
func (r *SQLiteRepository) Close() error {
	if r.db == nil {
		return nil
	}
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	r.logger.Debug().Msg("Database connection closed")
	return nil
}

// Initialize sets up the database schema (tables, indexes).
// Should be idempotent (safe to call multiple times).
func (r *SQLiteRepository) Initialize() error {
	r.logger.Debug().Msg("Initializing database schema")

	// Create tables in order (respecting foreign key dependencies)
	tables := []string{
		// Profiles table (no dependencies)
		`CREATE TABLE IF NOT EXISTS profiles (
			id TEXT PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			name TEXT,
			title TEXT,
			company TEXT,
			location TEXT,
			discovered_at TIMESTAMP NOT NULL,
			last_visited_at TIMESTAMP NOT NULL,
			visited_count INTEGER NOT NULL DEFAULT 1
		)`,

		// Connection requests table (depends on profiles)
		`CREATE TABLE IF NOT EXISTS connection_requests (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL REFERENCES profiles(id),
			profile_url TEXT NOT NULL,
			status TEXT NOT NULL,
			note TEXT,
			sent_at TIMESTAMP NOT NULL,
			responded_at TIMESTAMP
		)`,

		// Messages table (depends on profiles)
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL REFERENCES profiles(id),
			profile_url TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL,
			sent_at TIMESTAMP NOT NULL,
			template_id TEXT
		)`,

		// Daily counters table
		`CREATE TABLE IF NOT EXISTS daily_counters (
			date DATE PRIMARY KEY,
			connections_sent INTEGER NOT NULL DEFAULT 0,
			messages_sent INTEGER NOT NULL DEFAULT 0,
			last_connection_at TIMESTAMP,
			last_message_at TIMESTAMP
		)`,

		// Hourly counters table
		`CREATE TABLE IF NOT EXISTS hourly_counters (
			hour TIMESTAMP PRIMARY KEY,
			connections_sent INTEGER NOT NULL DEFAULT 0,
			messages_sent INTEGER NOT NULL DEFAULT 0
		)`,

		// Session state table
		`CREATE TABLE IF NOT EXISTS session_state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
	}

	// Create all tables
	for _, tableSQL := range tables {
		if _, err := r.db.Exec(tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_profiles_url ON profiles(url)",
		"CREATE INDEX IF NOT EXISTS idx_profiles_discovered_at ON profiles(discovered_at)",
		"CREATE INDEX IF NOT EXISTS idx_connection_requests_profile_id ON connection_requests(profile_id)",
		"CREATE INDEX IF NOT EXISTS idx_connection_requests_status ON connection_requests(status)",
		"CREATE INDEX IF NOT EXISTS idx_connection_requests_sent_at ON connection_requests(sent_at)",
		"CREATE INDEX IF NOT EXISTS idx_connection_requests_profile_url ON connection_requests(profile_url)",
		"CREATE INDEX IF NOT EXISTS idx_messages_profile_id ON messages(profile_id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status)",
		"CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at)",
		"CREATE INDEX IF NOT EXISTS idx_messages_profile_url ON messages(profile_url)",
	}

	// Create all indexes
	for _, indexSQL := range indexes {
		if _, err := r.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	r.logger.Info().Msg("Database schema initialized successfully")
	return nil
}

// Helper function to format time as RFC3339 string for SQLite
func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// Helper function to parse RFC3339 string from SQLite
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// Helper function to format date as YYYY-MM-DD
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// Helper function to format hour as YYYY-MM-DD HH:00:00
func formatHour(t time.Time) string {
	// Round down to the hour
	hour := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	return hour.Format("2006-01-02 15:00:00")
}

// Helper function to parse hour string
func parseHour(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:00:00", s)
}

// ============================================================================
// Profile Operations
// ============================================================================

// SaveProfile saves or updates a profile (upsert).
func (r *SQLiteRepository) SaveProfile(profile *models.Profile) error {
	r.logger.Debug().Str("url", profile.URL).Str("id", profile.ID).Msg("Saving profile")

	query := `INSERT OR REPLACE INTO profiles 
		(id, url, name, title, company, location, discovered_at, last_visited_at, visited_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		profile.ID,
		profile.URL,
		profile.Name,
		profile.Title,
		profile.Company,
		profile.Location,
		formatTime(profile.DiscoveredAt),
		formatTime(profile.LastVisitedAt),
		profile.VisitedCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save profile %s: %w", profile.URL, err)
	}

	return nil
}

// GetProfileByURL retrieves a profile by its URL.
// Returns nil, nil if not found (not an error).
func (r *SQLiteRepository) GetProfileByURL(url string) (*models.Profile, error) {
	r.logger.Debug().Str("url", url).Msg("Getting profile by URL")

	query := `SELECT id, url, name, title, company, location, discovered_at, last_visited_at, visited_count
		FROM profiles WHERE url = ?`

	var profile models.Profile
	var discoveredAtStr, lastVisitedAtStr string

	err := r.db.QueryRow(query, url).Scan(
		&profile.ID,
		&profile.URL,
		&profile.Name,
		&profile.Title,
		&profile.Company,
		&profile.Location,
		&discoveredAtStr,
		&lastVisitedAtStr,
		&profile.VisitedCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile by URL %s: %w", url, err)
	}

	// Parse timestamps
	discoveredAt, err := parseTime(discoveredAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discovered_at: %w", err)
	}
	profile.DiscoveredAt = discoveredAt

	lastVisitedAt, err := parseTime(lastVisitedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_visited_at: %w", err)
	}
	profile.LastVisitedAt = lastVisitedAt

	return &profile, nil
}

// GetProfileByID retrieves a profile by its ID.
// Returns nil, nil if not found (not an error).
func (r *SQLiteRepository) GetProfileByID(id string) (*models.Profile, error) {
	r.logger.Debug().Str("id", id).Msg("Getting profile by ID")

	query := `SELECT id, url, name, title, company, location, discovered_at, last_visited_at, visited_count
		FROM profiles WHERE id = ?`

	var profile models.Profile
	var discoveredAtStr, lastVisitedAtStr string

	err := r.db.QueryRow(query, id).Scan(
		&profile.ID,
		&profile.URL,
		&profile.Name,
		&profile.Title,
		&profile.Company,
		&profile.Location,
		&discoveredAtStr,
		&lastVisitedAtStr,
		&profile.VisitedCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile by ID %s: %w", id, err)
	}

	// Parse timestamps
	discoveredAt, err := parseTime(discoveredAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discovered_at: %w", err)
	}
	profile.DiscoveredAt = discoveredAt

	lastVisitedAt, err := parseTime(lastVisitedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_visited_at: %w", err)
	}
	profile.LastVisitedAt = lastVisitedAt

	return &profile, nil
}

// UpdateProfile updates an existing profile.
func (r *SQLiteRepository) UpdateProfile(profile *models.Profile) error {
	r.logger.Debug().Str("id", profile.ID).Msg("Updating profile")

	query := `UPDATE profiles SET
		name = ?, title = ?, company = ?, location = ?,
		last_visited_at = ?, visited_count = ?
		WHERE id = ?`

	_, err := r.db.Exec(query,
		profile.Name,
		profile.Title,
		profile.Company,
		profile.Location,
		formatTime(profile.LastVisitedAt),
		profile.VisitedCount,
		profile.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update profile %s: %w", profile.ID, err)
	}

	return nil
}

// HasVisitedProfile checks if a profile URL has been visited.
func (r *SQLiteRepository) HasVisitedProfile(url string) (bool, error) {
	r.logger.Debug().Str("url", url).Msg("Checking if profile visited")

	query := `SELECT COUNT(*) FROM profiles WHERE url = ?`
	var count int

	err := r.db.QueryRow(query, url).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check profile visit: %w", err)
	}

	return count > 0, nil
}

// GetAllProfiles retrieves all profiles, ordered by discovery date (newest first).
func (r *SQLiteRepository) GetAllProfiles() ([]*models.Profile, error) {
	r.logger.Debug().Msg("Getting all profiles")

	query := `SELECT id, url, name, title, company, location, discovered_at, last_visited_at, visited_count
		FROM profiles ORDER BY discovered_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*models.Profile
	for rows.Next() {
		var profile models.Profile
		var discoveredAtStr, lastVisitedAtStr string

		err := rows.Scan(
			&profile.ID,
			&profile.URL,
			&profile.Name,
			&profile.Title,
			&profile.Company,
			&profile.Location,
			&discoveredAtStr,
			&lastVisitedAtStr,
			&profile.VisitedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}

		// Parse timestamps
		discoveredAt, err := parseTime(discoveredAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse discovered_at: %w", err)
		}
		profile.DiscoveredAt = discoveredAt

		lastVisitedAt, err := parseTime(lastVisitedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_visited_at: %w", err)
		}
		profile.LastVisitedAt = lastVisitedAt

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profiles: %w", err)
	}

	return profiles, nil
}

// ============================================================================
// ConnectionRequest Operations
// ============================================================================

// SaveConnectionRequest saves a connection request.
func (r *SQLiteRepository) SaveConnectionRequest(req *models.ConnectionRequest) error {
	r.logger.Debug().Str("id", req.ID).Str("profile_url", req.ProfileURL).Msg("Saving connection request")

	query := `INSERT INTO connection_requests 
		(id, profile_id, profile_url, status, note, sent_at, responded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	var respondedAtStr interface{}
	if req.RespondedAt != nil {
		respondedAtStr = formatTime(*req.RespondedAt)
	}

	_, err := r.db.Exec(query,
		req.ID,
		req.ProfileID,
		req.ProfileURL,
		req.Status,
		req.Note,
		formatTime(req.SentAt),
		respondedAtStr,
	)

	if err != nil {
		return fmt.Errorf("failed to save connection request %s: %w", req.ID, err)
	}

	return nil
}

// GetConnectionRequestByID retrieves a connection request by its ID.
// Returns nil, nil if not found (not an error).
func (r *SQLiteRepository) GetConnectionRequestByID(id string) (*models.ConnectionRequest, error) {
	r.logger.Debug().Str("id", id).Msg("Getting connection request by ID")

	query := `SELECT id, profile_id, profile_url, status, note, sent_at, responded_at
		FROM connection_requests WHERE id = ?`

	var req models.ConnectionRequest
	var sentAtStr string
	var respondedAtStr sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&req.ID,
		&req.ProfileID,
		&req.ProfileURL,
		&req.Status,
		&req.Note,
		&sentAtStr,
		&respondedAtStr,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connection request by ID %s: %w", id, err)
	}

	// Parse timestamps
	sentAt, err := parseTime(sentAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sent_at: %w", err)
	}
	req.SentAt = sentAt

	if respondedAtStr.Valid {
		respondedAt, err := parseTime(respondedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse responded_at: %w", err)
		}
		req.RespondedAt = &respondedAt
	}

	return &req, nil
}

// GetConnectionRequestsByProfileID retrieves all connection requests for a profile.
func (r *SQLiteRepository) GetConnectionRequestsByProfileID(profileID string) ([]*models.ConnectionRequest, error) {
	r.logger.Debug().Str("profile_id", profileID).Msg("Getting connection requests by profile ID")

	query := `SELECT id, profile_id, profile_url, status, note, sent_at, responded_at
		FROM connection_requests WHERE profile_id = ? ORDER BY sent_at DESC`

	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.ConnectionRequest
	for rows.Next() {
		var req models.ConnectionRequest
		var sentAtStr string
		var respondedAtStr sql.NullString

		err := rows.Scan(
			&req.ID,
			&req.ProfileID,
			&req.ProfileURL,
			&req.Status,
			&req.Note,
			&sentAtStr,
			&respondedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection request: %w", err)
		}

		// Parse timestamps
		sentAt, err := parseTime(sentAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sent_at: %w", err)
		}
		req.SentAt = sentAt

		if respondedAtStr.Valid {
			respondedAt, err := parseTime(respondedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse responded_at: %w", err)
			}
			req.RespondedAt = &respondedAt
		}

		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection requests: %w", err)
	}

	return requests, nil
}

// UpdateConnectionRequest updates an existing connection request.
func (r *SQLiteRepository) UpdateConnectionRequest(req *models.ConnectionRequest) error {
	r.logger.Debug().Str("id", req.ID).Msg("Updating connection request")

	query := `UPDATE connection_requests SET
		status = ?, note = ?, responded_at = ?
		WHERE id = ?`

	var respondedAtStr interface{}
	if req.RespondedAt != nil {
		respondedAtStr = formatTime(*req.RespondedAt)
	}

	_, err := r.db.Exec(query,
		req.Status,
		req.Note,
		respondedAtStr,
		req.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update connection request %s: %w", req.ID, err)
	}

	return nil
}

// CountConnectionRequestsToday counts connection requests sent today.
func (r *SQLiteRepository) CountConnectionRequestsToday() (int, error) {
	r.logger.Debug().Msg("Counting connection requests today")

	query := `SELECT COUNT(*) FROM connection_requests 
		WHERE DATE(sent_at) = DATE('now')`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count connection requests today: %w", err)
	}

	return count, nil
}

// CountConnectionRequestsInHour counts connection requests sent in a specific hour.
func (r *SQLiteRepository) CountConnectionRequestsInHour(hour time.Time) (int, error) {
	r.logger.Debug().Str("hour", formatHour(hour)).Msg("Counting connection requests in hour")

	hourStr := formatHour(hour)
	query := `SELECT COUNT(*) FROM connection_requests 
		WHERE strftime('%Y-%m-%d %H:00:00', sent_at) = ?`

	var count int
	err := r.db.QueryRow(query, hourStr).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count connection requests in hour: %w", err)
	}

	return count, nil
}

// GetConnectionRequestsByStatus retrieves all connection requests with a specific status.
func (r *SQLiteRepository) GetConnectionRequestsByStatus(status string) ([]*models.ConnectionRequest, error) {
	r.logger.Debug().Str("status", status).Msg("Getting connection requests by status")

	query := `SELECT id, profile_id, profile_url, status, note, sent_at, responded_at
		FROM connection_requests WHERE status = ? ORDER BY sent_at DESC`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.ConnectionRequest
	for rows.Next() {
		var req models.ConnectionRequest
		var sentAtStr string
		var respondedAtStr sql.NullString

		err := rows.Scan(
			&req.ID,
			&req.ProfileID,
			&req.ProfileURL,
			&req.Status,
			&req.Note,
			&sentAtStr,
			&respondedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection request: %w", err)
		}

		// Parse timestamps
		sentAt, err := parseTime(sentAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sent_at: %w", err)
		}
		req.SentAt = sentAt

		if respondedAtStr.Valid {
			respondedAt, err := parseTime(respondedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse responded_at: %w", err)
			}
			req.RespondedAt = &respondedAt
		}

		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection requests: %w", err)
	}

	return requests, nil
}

// GetPendingConnectionRequests retrieves all pending connection requests.
func (r *SQLiteRepository) GetPendingConnectionRequests() ([]*models.ConnectionRequest, error) {
	return r.GetConnectionRequestsByStatus("pending")
}

// GetAcceptedConnectionRequests retrieves all accepted connection requests.
func (r *SQLiteRepository) GetAcceptedConnectionRequests() ([]*models.ConnectionRequest, error) {
	return r.GetConnectionRequestsByStatus("accepted")
}

// ============================================================================
// Message Operations
// ============================================================================

// SaveMessage saves a message.
func (r *SQLiteRepository) SaveMessage(msg *models.Message) error {
	r.logger.Debug().Str("id", msg.ID).Str("profile_url", msg.ProfileURL).Msg("Saving message")

	query := `INSERT INTO messages 
		(id, profile_id, profile_url, content, status, sent_at, template_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		msg.ID,
		msg.ProfileID,
		msg.ProfileURL,
		msg.Content,
		msg.Status,
		formatTime(msg.SentAt),
		msg.TemplateID,
	)

	if err != nil {
		return fmt.Errorf("failed to save message %s: %w", msg.ID, err)
	}

	return nil
}

// GetMessageByID retrieves a message by its ID.
// Returns nil, nil if not found (not an error).
func (r *SQLiteRepository) GetMessageByID(id string) (*models.Message, error) {
	r.logger.Debug().Str("id", id).Msg("Getting message by ID")

	query := `SELECT id, profile_id, profile_url, content, status, sent_at, template_id
		FROM messages WHERE id = ?`

	var msg models.Message
	var sentAtStr string

	err := r.db.QueryRow(query, id).Scan(
		&msg.ID,
		&msg.ProfileID,
		&msg.ProfileURL,
		&msg.Content,
		&msg.Status,
		&sentAtStr,
		&msg.TemplateID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message by ID %s: %w", id, err)
	}

	// Parse timestamp
	sentAt, err := parseTime(sentAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sent_at: %w", err)
	}
	msg.SentAt = sentAt

	return &msg, nil
}

// GetMessagesByProfileID retrieves all messages for a profile.
func (r *SQLiteRepository) GetMessagesByProfileID(profileID string) ([]*models.Message, error) {
	r.logger.Debug().Str("profile_id", profileID).Msg("Getting messages by profile ID")

	query := `SELECT id, profile_id, profile_url, content, status, sent_at, template_id
		FROM messages WHERE profile_id = ? ORDER BY sent_at DESC`

	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		var sentAtStr string

		err := rows.Scan(
			&msg.ID,
			&msg.ProfileID,
			&msg.ProfileURL,
			&msg.Content,
			&msg.Status,
			&sentAtStr,
			&msg.TemplateID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse timestamp
		sentAt, err := parseTime(sentAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sent_at: %w", err)
		}
		msg.SentAt = sentAt

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// UpdateMessage updates an existing message.
func (r *SQLiteRepository) UpdateMessage(msg *models.Message) error {
	r.logger.Debug().Str("id", msg.ID).Msg("Updating message")

	query := `UPDATE messages SET
		content = ?, status = ?, template_id = ?
		WHERE id = ?`

	_, err := r.db.Exec(query,
		msg.Content,
		msg.Status,
		msg.TemplateID,
		msg.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update message %s: %w", msg.ID, err)
	}

	return nil
}

// CountMessagesToday counts messages sent today.
func (r *SQLiteRepository) CountMessagesToday() (int, error) {
	r.logger.Debug().Msg("Counting messages today")

	query := `SELECT COUNT(*) FROM messages 
		WHERE DATE(sent_at) = DATE('now')`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages today: %w", err)
	}

	return count, nil
}

// CountMessagesInHour counts messages sent in a specific hour.
func (r *SQLiteRepository) CountMessagesInHour(hour time.Time) (int, error) {
	r.logger.Debug().Str("hour", formatHour(hour)).Msg("Counting messages in hour")

	hourStr := formatHour(hour)
	query := `SELECT COUNT(*) FROM messages 
		WHERE strftime('%Y-%m-%d %H:00:00', sent_at) = ?`

	var count int
	err := r.db.QueryRow(query, hourStr).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages in hour: %w", err)
	}

	return count, nil
}

// GetAllMessages retrieves all messages, ordered by sent date (newest first).
func (r *SQLiteRepository) GetAllMessages() ([]*models.Message, error) {
	r.logger.Debug().Msg("Getting all messages")

	query := `SELECT id, profile_id, profile_url, content, status, sent_at, template_id
		FROM messages ORDER BY sent_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		var sentAtStr string

		err := rows.Scan(
			&msg.ID,
			&msg.ProfileID,
			&msg.ProfileURL,
			&msg.Content,
			&msg.Status,
			&sentAtStr,
			&msg.TemplateID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse timestamp
		sentAt, err := parseTime(sentAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sent_at: %w", err)
		}
		msg.SentAt = sentAt

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// ============================================================================
// DailyCounters Operations
// ============================================================================

// GetDailyCounters retrieves daily counters for a specific date.
// Returns a new DailyCounters with zeros if not found.
func (r *SQLiteRepository) GetDailyCounters(date time.Time) (*models.DailyCounters, error) {
	r.logger.Debug().Str("date", formatDate(date)).Msg("Getting daily counters")

	dateStr := formatDate(date)
	query := `SELECT date, connections_sent, messages_sent, last_connection_at, last_message_at
		FROM daily_counters WHERE date = ?`

	var counters models.DailyCounters
	var lastConnectionAtStr, lastMessageAtStr sql.NullString

	err := r.db.QueryRow(query, dateStr).Scan(
		&dateStr, // We'll parse it back
		&counters.ConnectionsSent,
		&counters.MessagesSent,
		&lastConnectionAtStr,
		&lastMessageAtStr,
	)

	if err == sql.ErrNoRows {
		// Return new counters with zeros
		dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		return &models.DailyCounters{
			Date:             dateOnly,
			ConnectionsSent:  0,
			MessagesSent:    0,
			LastConnectionAt: time.Time{},
			LastMessageAt:    time.Time{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily counters: %w", err)
	}

	// Parse date
	dateOnly, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	counters.Date = dateOnly

	// Parse nullable timestamps
	if lastConnectionAtStr.Valid {
		lastConnectionAt, err := parseTime(lastConnectionAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_connection_at: %w", err)
		}
		counters.LastConnectionAt = lastConnectionAt
	}

	if lastMessageAtStr.Valid {
		lastMessageAt, err := parseTime(lastMessageAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_message_at: %w", err)
		}
		counters.LastMessageAt = lastMessageAt
	}

	return &counters, nil
}

// UpdateDailyCounters updates daily counters (upsert).
func (r *SQLiteRepository) UpdateDailyCounters(counters *models.DailyCounters) error {
	r.logger.Debug().Str("date", formatDate(counters.Date)).Msg("Updating daily counters")

	query := `INSERT OR REPLACE INTO daily_counters 
		(date, connections_sent, messages_sent, last_connection_at, last_message_at)
		VALUES (?, ?, ?, ?, ?)`

	var lastConnectionAtStr, lastMessageAtStr interface{}
	if !counters.LastConnectionAt.IsZero() {
		lastConnectionAtStr = formatTime(counters.LastConnectionAt)
	}
	if !counters.LastMessageAt.IsZero() {
		lastMessageAtStr = formatTime(counters.LastMessageAt)
	}

	_, err := r.db.Exec(query,
		formatDate(counters.Date),
		counters.ConnectionsSent,
		counters.MessagesSent,
		lastConnectionAtStr,
		lastMessageAtStr,
	)

	if err != nil {
		return fmt.Errorf("failed to update daily counters: %w", err)
	}

	return nil
}

// IncrementConnectionCount increments the connection count for a date.
func (r *SQLiteRepository) IncrementConnectionCount(date time.Time) error {
	r.logger.Debug().Str("date", formatDate(date)).Msg("Incrementing connection count")

	// Get current counters
	counters, err := r.GetDailyCounters(date)
	if err != nil {
		return fmt.Errorf("failed to get daily counters: %w", err)
	}

	// Increment and update
	counters.ConnectionsSent++
	counters.LastConnectionAt = time.Now()

	return r.UpdateDailyCounters(counters)
}

// IncrementMessageCount increments the message count for a date.
func (r *SQLiteRepository) IncrementMessageCount(date time.Time) error {
	r.logger.Debug().Str("date", formatDate(date)).Msg("Incrementing message count")

	// Get current counters
	counters, err := r.GetDailyCounters(date)
	if err != nil {
		return fmt.Errorf("failed to get daily counters: %w", err)
	}

	// Increment and update
	counters.MessagesSent++
	counters.LastMessageAt = time.Now()

	return r.UpdateDailyCounters(counters)
}

// ResetDailyCountersIfNeeded ensures daily counters exist for today.
// Creates new row with zeros if it doesn't exist.
func (r *SQLiteRepository) ResetDailyCountersIfNeeded(date time.Time) error {
	r.logger.Debug().Str("date", formatDate(date)).Msg("Resetting daily counters if needed")

	// Try to get counters
	counters, err := r.GetDailyCounters(date)
	if err != nil {
		return fmt.Errorf("failed to get daily counters: %w", err)
	}

	// If counters are zero (new day), ensure they're saved
	if counters.ConnectionsSent == 0 && counters.MessagesSent == 0 {
		return r.UpdateDailyCounters(counters)
	}

	return nil
}

// ============================================================================
// HourlyCounters Operations
// ============================================================================

// GetHourlyCounters retrieves hourly counters for a specific hour.
// Returns a new HourlyCounters with zeros if not found.
func (r *SQLiteRepository) GetHourlyCounters(hour time.Time) (*models.HourlyCounters, error) {
	r.logger.Debug().Str("hour", formatHour(hour)).Msg("Getting hourly counters")

	hourStr := formatHour(hour)
	query := `SELECT hour, connections_sent, messages_sent
		FROM hourly_counters WHERE hour = ?`

	var counters models.HourlyCounters
	var hourStrResult string

	err := r.db.QueryRow(query, hourStr).Scan(
		&hourStrResult,
		&counters.ConnectionsSent,
		&counters.MessagesSent,
	)

	if err == sql.ErrNoRows {
		// Return new counters with zeros
		hourOnly := time.Date(hour.Year(), hour.Month(), hour.Day(), hour.Hour(), 0, 0, 0, hour.Location())
		return &models.HourlyCounters{
			Hour:            hourOnly,
			ConnectionsSent: 0,
			MessagesSent:    0,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly counters: %w", err)
	}

	// Parse hour
	hourOnly, err := parseHour(hourStrResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hour: %w", err)
	}
	counters.Hour = hourOnly

	return &counters, nil
}

// UpdateHourlyCounters updates hourly counters (upsert).
func (r *SQLiteRepository) UpdateHourlyCounters(counters *models.HourlyCounters) error {
	r.logger.Debug().Str("hour", formatHour(counters.Hour)).Msg("Updating hourly counters")

	query := `INSERT OR REPLACE INTO hourly_counters 
		(hour, connections_sent, messages_sent)
		VALUES (?, ?, ?)`

	_, err := r.db.Exec(query,
		formatHour(counters.Hour),
		counters.ConnectionsSent,
		counters.MessagesSent,
	)

	if err != nil {
		return fmt.Errorf("failed to update hourly counters: %w", err)
	}

	return nil
}

// IncrementHourlyConnectionCount increments the connection count for an hour.
func (r *SQLiteRepository) IncrementHourlyConnectionCount(hour time.Time) error {
	r.logger.Debug().Str("hour", formatHour(hour)).Msg("Incrementing hourly connection count")

	// Get current counters
	counters, err := r.GetHourlyCounters(hour)
	if err != nil {
		return fmt.Errorf("failed to get hourly counters: %w", err)
	}

	// Increment and update
	counters.ConnectionsSent++

	return r.UpdateHourlyCounters(counters)
}

// IncrementHourlyMessageCount increments the message count for an hour.
func (r *SQLiteRepository) IncrementHourlyMessageCount(hour time.Time) error {
	r.logger.Debug().Str("hour", formatHour(hour)).Msg("Incrementing hourly message count")

	// Get current counters
	counters, err := r.GetHourlyCounters(hour)
	if err != nil {
		return fmt.Errorf("failed to get hourly counters: %w", err)
	}

	// Increment and update
	counters.MessagesSent++

	return r.UpdateHourlyCounters(counters)
}

// ============================================================================
// SessionState Operations
// ============================================================================

// GetSessionState retrieves session state by key.
// Returns nil, nil if not found (not an error).
func (r *SQLiteRepository) GetSessionState(key string) (*models.SessionState, error) {
	r.logger.Debug().Str("key", key).Msg("Getting session state")

	query := `SELECT key, value, updated_at FROM session_state WHERE key = ?`

	var state models.SessionState
	var updatedAtStr string

	err := r.db.QueryRow(query, key).Scan(
		&state.Key,
		&state.Value,
		&updatedAtStr,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session state %s: %w", key, err)
	}

	// Parse timestamp
	updatedAt, err := parseTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}
	state.UpdatedAt = updatedAt

	return &state, nil
}

// SetSessionState sets session state (upsert).
func (r *SQLiteRepository) SetSessionState(key, value string) error {
	r.logger.Debug().Str("key", key).Msg("Setting session state")

	query := `INSERT OR REPLACE INTO session_state (key, value, updated_at)
		VALUES (?, ?, ?)`

	_, err := r.db.Exec(query, key, value, formatTime(time.Now()))
	if err != nil {
		return fmt.Errorf("failed to set session state %s: %w", key, err)
	}

	return nil
}

// GetLastRunTime retrieves the last run time from session state.
// Returns zero time if not found.
func (r *SQLiteRepository) GetLastRunTime() (time.Time, error) {
	state, err := r.GetSessionState("last_run")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last run time: %w", err)
	}

	if state == nil {
		return time.Time{}, nil
	}

	// Parse value as timestamp
	lastRun, err := parseTime(state.Value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse last run time: %w", err)
	}

	return lastRun, nil
}

// SetLastRunTime sets the last run time in session state.
func (r *SQLiteRepository) SetLastRunTime(t time.Time) error {
	return r.SetSessionState("last_run", formatTime(t))
}

