package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hansel06/linkedin-automation/internal/config"
	"github.com/hansel06/linkedin-automation/internal/logging"
	"github.com/hansel06/linkedin-automation/internal/models"
	"github.com/hansel06/linkedin-automation/internal/storage"
)

func main() {
	fmt.Println("=== Storage Implementation Test ===")
	fmt.Println()

	// Step 1: Load config (use defaults for testing)
	fmt.Println("Step 1: Loading configuration...")
	cfg := config.NewDefault()
	fmt.Println("✓ Configuration loaded")
	fmt.Printf("  - Database Path: %s\n", cfg.Storage.DatabasePath)
	fmt.Println()

	// Step 2: Create logger
	fmt.Println("Step 2: Creating logger...")
	logger := logging.NewLogger(cfg.Logging.LogLevel, cfg.Logging.LogFormat)
	logger = logger.WithComponent("test-storage")
	fmt.Println("✓ Logger created")
	fmt.Println()

	// Step 3: Create repository
	fmt.Println("Step 3: Creating SQLite repository...")
	repo, err := storage.NewSQLiteRepository(cfg.Storage.DatabasePath, logger)
	if err != nil {
		fmt.Printf("✗ Failed to create repository: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Repository created")
	fmt.Println()

	// Ensure repository is closed at the end
	defer func() {
		fmt.Println()
		fmt.Println("Step 12: Closing repository...")
		if err := repo.Close(); err != nil {
			fmt.Printf("✗ Failed to close repository: %v\n", err)
		} else {
			fmt.Println("✓ Repository closed successfully")
		}
	}()

	// Step 4: Initialize database schema
	fmt.Println("Step 4: Initializing database schema...")
	if err := repo.Initialize(); err != nil {
		fmt.Printf("✗ Failed to initialize schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Schema initialized")
	fmt.Println()

	// Step 5: Test Profile operations
	fmt.Println("Step 5: Testing Profile operations...")
	
	// Create a test profile
	profile := models.NewProfile("https://linkedin.com/in/test-user")
	profile.Name = "Test User"
	profile.Title = "Software Engineer"
	profile.Company = "Test Company"
	profile.Location = "Test Location"
	
	if err := repo.SaveProfile(profile); err != nil {
		fmt.Printf("✗ Failed to save profile: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Profile saved")
	
	// Retrieve by URL
	retrieved, err := repo.GetProfileByURL(profile.URL)
	if err != nil {
		fmt.Printf("✗ Failed to get profile by URL: %v\n", err)
		os.Exit(1)
	}
	if retrieved == nil {
		fmt.Printf("✗ Profile not found by URL\n")
		os.Exit(1)
	}
	fmt.Printf("✓ Profile retrieved by URL: %s\n", retrieved.Name)
	
	// Update profile
	retrieved.VisitedCount = 5
	retrieved.LastVisitedAt = time.Now()
	if err := repo.UpdateProfile(retrieved); err != nil {
		fmt.Printf("✗ Failed to update profile: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Profile updated")
	
	// Check if visited
	hasVisited, err := repo.HasVisitedProfile(profile.URL)
	if err != nil {
		fmt.Printf("✗ Failed to check visit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Has visited check: %v\n", hasVisited)
	fmt.Println()

	// Step 6: Test ConnectionRequest operations
	fmt.Println("Step 6: Testing ConnectionRequest operations...")
	
	connReq := models.NewConnectionRequest(profile.ID, profile.URL, "Hello, let's connect!")
	if err := repo.SaveConnectionRequest(connReq); err != nil {
		fmt.Printf("✗ Failed to save connection request: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Connection request saved")
	
	// Count today
	count, err := repo.CountConnectionRequestsToday()
	if err != nil {
		fmt.Printf("✗ Failed to count today: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Connection requests today: %d\n", count)
	
	// Get pending
	pending, err := repo.GetPendingConnectionRequests()
	if err != nil {
		fmt.Printf("✗ Failed to get pending: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Pending connection requests: %d\n", len(pending))
	
	// Update status
	connReq.Status = "accepted"
	now := time.Now()
	connReq.RespondedAt = &now
	if err := repo.UpdateConnectionRequest(connReq); err != nil {
		fmt.Printf("✗ Failed to update connection request: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Connection request updated to accepted")
	fmt.Println()

	// Step 7: Test Message operations
	fmt.Println("Step 7: Testing Message operations...")
	
	message := models.NewMessage(profile.ID, profile.URL, "Thanks for connecting!", "template-1")
	if err := repo.SaveMessage(message); err != nil {
		fmt.Printf("✗ Failed to save message: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Message saved")
	
	// Count today
	msgCount, err := repo.CountMessagesToday()
	if err != nil {
		fmt.Printf("✗ Failed to count messages: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Messages today: %d\n", msgCount)
	
	// Get all messages
	allMessages, err := repo.GetAllMessages()
	if err != nil {
		fmt.Printf("✗ Failed to get all messages: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Total messages: %d\n", len(allMessages))
	fmt.Println()

	// Step 8: Test DailyCounters operations
	fmt.Println("Step 8: Testing DailyCounters operations...")
	
	today := time.Now()
	counters, err := repo.GetDailyCounters(today)
	if err != nil {
		fmt.Printf("✗ Failed to get daily counters: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Daily counters retrieved: connections=%d, messages=%d\n", 
		counters.ConnectionsSent, counters.MessagesSent)
	
	// Increment connection count
	if err := repo.IncrementConnectionCount(today); err != nil {
		fmt.Printf("✗ Failed to increment connection count: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Connection count incremented")
	
	// Get updated counters
	counters, err = repo.GetDailyCounters(today)
	if err != nil {
		fmt.Printf("✗ Failed to get updated counters: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Updated counters: connections=%d\n", counters.ConnectionsSent)
	fmt.Println()

	// Step 9: Test HourlyCounters operations
	fmt.Println("Step 9: Testing HourlyCounters operations...")
	
	currentHour := time.Now()
	hourlyCounters, err := repo.GetHourlyCounters(currentHour)
	if err != nil {
		fmt.Printf("✗ Failed to get hourly counters: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Hourly counters retrieved: connections=%d, messages=%d\n",
		hourlyCounters.ConnectionsSent, hourlyCounters.MessagesSent)
	
	// Increment hourly connection count
	if err := repo.IncrementHourlyConnectionCount(currentHour); err != nil {
		fmt.Printf("✗ Failed to increment hourly connection count: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Hourly connection count incremented")
	
	// Count in hour
	hourCount, err := repo.CountConnectionRequestsInHour(currentHour)
	if err != nil {
		fmt.Printf("✗ Failed to count in hour: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Connection requests in current hour: %d\n", hourCount)
	fmt.Println()

	// Step 10: Test SessionState operations
	fmt.Println("Step 10: Testing SessionState operations...")
	
	// Set session state
	if err := repo.SetSessionState("test_key", "test_value"); err != nil {
		fmt.Printf("✗ Failed to set session state: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Session state set")
	
	// Get session state
	state, err := repo.GetSessionState("test_key")
	if err != nil {
		fmt.Printf("✗ Failed to get session state: %v\n", err)
		os.Exit(1)
	}
	if state == nil {
		fmt.Printf("✗ Session state not found\n")
		os.Exit(1)
	}
	fmt.Printf("✓ Session state retrieved: key=%s, value=%s\n", state.Key, state.Value)
	
	// Test last run time
	lastRun := time.Now()
	if err := repo.SetLastRunTime(lastRun); err != nil {
		fmt.Printf("✗ Failed to set last run time: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Last run time set")
	
	retrievedLastRun, err := repo.GetLastRunTime()
	if err != nil {
		fmt.Printf("✗ Failed to get last run time: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Last run time retrieved: %v\n", retrievedLastRun.Format(time.RFC3339))
	fmt.Println()

	// Step 11: Test GetAllProfiles
	fmt.Println("Step 11: Testing GetAllProfiles...")
	allProfiles, err := repo.GetAllProfiles()
	if err != nil {
		fmt.Printf("✗ Failed to get all profiles: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Total profiles: %d\n", len(allProfiles))
	if len(allProfiles) > 0 {
		fmt.Printf("  - First profile: %s (%s)\n", allProfiles[0].Name, allProfiles[0].URL)
	}
	fmt.Println()

	fmt.Println("=== All Tests Completed Successfully! ===")
	fmt.Println()
	fmt.Println("The storage implementation is working correctly!")
	fmt.Println("You can inspect the database file at:", cfg.Storage.DatabasePath)
	fmt.Println("Use a SQLite browser tool to view the data.")
}

