# Storage Implementation Test

This document describes how to test the Phase 8 storage implementation.

## Quick Test

Run the test program:

```bash
go run ./cmd/test-storage
```

Or build and run:

```bash
go build -o test-storage.exe ./cmd/test-storage
./test-storage.exe
```

## What the Test Does

The test program verifies:

1. ✅ **Configuration Loading** - Loads default config
2. ✅ **Logger Creation** - Creates structured logger
3. ✅ **Repository Creation** - Creates SQLite repository instance
4. ✅ **Schema Initialization** - Creates all tables and indexes
5. ✅ **Profile Operations** - Tests save, get, update, has visited
6. ✅ **ConnectionRequest Operations** - Tests save, count, update, status filtering
7. ✅ **Message Operations** - Tests save, count, get all
8. ✅ **DailyCounters Operations** - Tests get, increment, update
9. ✅ **HourlyCounters Operations** - Tests get, increment, count in hour
10. ✅ **SessionState Operations** - Tests set, get, last run time
11. ✅ **GetAllProfiles** - Tests retrieving all profiles
12. ✅ **Repository Cleanup** - Closes database connection

## Expected Behavior

When you run the test:

1. Database file will be created at `data/automation.db`
2. All tables and indexes will be created automatically
3. Test data will be inserted (1 profile, 1 connection request, 1 message)
4. You should see console output showing each step with ✓ checkmarks
5. Counters will be incremented and verified
6. Database connection will be closed at the end

## Database Inspection

After running the test, you can inspect the database using any SQLite browser tool:

### Recommended Tools:
- **DB Browser for SQLite** (https://sqlitebrowser.org/) - Free, cross-platform
- **SQLiteStudio** (https://sqlitestudio.pl/) - Free, cross-platform
- **VS Code Extension**: SQLite Viewer

### What to Check:

1. **profiles table**: Should have 1 test profile
2. **connection_requests table**: Should have 1 connection request (status: "accepted")
3. **messages table**: Should have 1 message
4. **daily_counters table**: Should have today's date with counters > 0
5. **hourly_counters table**: Should have current hour with counters > 0
6. **session_state table**: Should have "test_key" and "last_run" entries

## Troubleshooting

### Database file not created
- Check if `data/` directory exists or can be created
- Check file permissions
- Check disk space

### Schema initialization fails
- Check if database file is locked (close SQLite browser if open)
- Check file permissions
- Delete `data/automation.db` and try again

### Foreign key constraint errors
- Ensure tables are created in order (profiles first, then connection_requests/messages)
- Check that profile IDs exist before creating connection requests/messages

### Time formatting errors
- Ensure system time is correct
- Check timezone settings

## Test Results

If all steps show ✓ (checkmark), the storage implementation is working correctly!

## Manual Verification

You can also manually verify the database:

```bash
# Using sqlite3 command-line tool (if installed)
sqlite3 data/automation.db

# Then run SQL queries:
.tables                    # List all tables
SELECT * FROM profiles;    # View all profiles
SELECT * FROM connection_requests;  # View all connection requests
SELECT * FROM messages;   # View all messages
SELECT * FROM daily_counters;  # View daily counters
SELECT * FROM hourly_counters;  # View hourly counters
SELECT * FROM session_state;  # View session state
```

## Next Steps

After verifying storage works:
- Phase 9: Implement authentication, search, connect, and message features
- These features will use the storage repository to persist state
- Rate limiting will use daily/hourly counters
- Duplicate prevention will use profile/request tracking

