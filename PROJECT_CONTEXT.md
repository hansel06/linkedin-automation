# Project Context

Purpose: Proof-of-concept LinkedIn automation in Go (Rod) to demonstrate human-like behavior and anti-detection, not a production bot.

Core capabilities:
- Authentication: login via env vars; detect login failures/checkpoints (2FA/captcha); persist/reuse cookies.
- Search/targeting: search by job/company/location/keywords; paginate; collect profile URLs; avoid duplicates.
- Connection requests: open profiles; click Connect; optional personalized note (char limit); track sent requests; enforce daily limits/cooldowns.
- Messaging: detect accepted connections; send follow-up messages; templates with dynamic variables; track message history/status.

Anti-detection (highest priority):
- Mandatory: human-like mouse movement (Bezier/curves, overshoot, micro-corrections, variable speed); randomized timing (think time, scroll speed, interaction intervals); browser fingerprint masking (UA, viewport, disable navigator.webdriver, randomize props).
- Choose at least 5 more: random scrolling; realistic typing (variable intervals, typos/backspace); hover/wander; activity scheduling (business hours/breaks); rate limiting/throttling; other subtle noise patterns.

Code quality expectations:
- Go-only for core automation; use Rod for browser control.
- Modular architecture (auth/search/connect/message/stealth/browser/config/storage/logging).
- Robust error handling, retries with backoff, graceful degradation.
- Structured logging with levels, timestamps, contextual fields.
- Config via YAML/JSON with env overrides and validation; sensible defaults.
- State persistence (SQLite or JSON) to track sent requests, accepted connections, messages, daily counters; support resume after interruption.
- Documentation: README, inline comments for complex logic, .env.example, setup/run/troubleshooting. Demo video showing setup and key flows.

Evaluation emphasis:
- Anti-detection quality > automation correctness > architecture > practical implementation. Stealth quality is critical.

Constraints:
- No ML/AI models; human-like behavior is deterministic + randomness.
- Acknowledge LinkedIn ToS violation; educational/PoC only; never production.

