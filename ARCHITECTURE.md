# Architecture Plan

Guiding rule: All browser interactions go through the stealth/human-behavior layer. No direct Rod calls from feature code.

Top-level layout (proposed):
- cmd/app/main.go           — wire config, logger, storage, browser, services; run orchestrations.
- internal/config           — load YAML/JSON, env overrides, defaults, validation.
- internal/logging          — structured logging with levels and contextual fields.
- internal/browser          — Rod setup, fingerprint masking (UA, viewport, navigator flags), page/session management.
- internal/stealth          — human-behavior wrapper: mouse movement (Bezier/curves, overshoot), hover, scroll, typing simulation, randomized timing, activity scheduling/rate limits hooks.
- internal/auth             — login flow, cookie reuse/persistence, checkpoint detection.
- internal/search           — build search queries, navigate results, parse profile URLs, pagination, deduplication.
- internal/connect          — visit profiles, click Connect, add note within limits, enforce daily quotas, record sent requests.
- internal/message          — detect accepted connections, send follow-up messages with templates/variables, track status/history.
- internal/storage          — persistence (SQLite or JSON) for state: sent requests, accepted connections, messages, counters, last run; simple repository interfaces.
- internal/models           — shared data structs (profile, message job, limits).
- internal/orchestrator     — high-level flows composing services with rate limiting/scheduling.
- pkg/utils (optional)      — small shared helpers if needed (avoid dumping ground).

Behavior/data flow:
Orchestrator → Services (auth/search/connect/message) → Stealth → Browser
State/limits consulted via storage/config; logging everywhere.

Stealth techniques to embed:
- Mandatory: human-like mouse movement; randomized timing; browser fingerprint masking.
- Additional (pick ≥5): random scrolling; realistic typing (variable intervals, typos/backspace); hover/wander; activity scheduling; rate limiting/throttling; occasional micro-pauses/scroll-backs.

Error handling:
- Explicit error returns; contextual wrapping.
- Retries with exponential backoff where sensible (network/nav actions).
- Graceful degradation and clear logging on failure paths.

Config surface (examples):
- Credentials from env; headless toggle; timeouts; delays jitter ranges; UA/viewport ranges; daily/hourly limits; schedule windows; storage path; log level/format.

State persistence:
- Durable store (SQLite or JSON) for: visited URLs, sent requests, accepted connections, sent messages, daily counters, last run timestamp, cookie jar.

Testing/observability:
- Unit where reasonable (pure logic).
- Structured logs to understand failures; optional dry-run/simulate mode for demo.

