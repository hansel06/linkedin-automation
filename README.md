# LinkedIn Automation (PoC)

This is a Go-based proof-of-concept demonstrating browser automation with human-like behavior and anti-detection techniques for LinkedIn. Educational/evaluation only; do not use in production.

## Status
- Planning/scaffolding in progress (no features implemented yet).

## Tech
- Go 1.22+ (using Rod for browser automation)
- Structured logging, YAML config, SQLite state (planned)

## Setup (planned)
1) Install Go.
2) Clone the repo.
3) Copy `env.example` to `.env` and fill credentials (never commit real secrets).
4) Install deps: `go mod tidy` (after we add them).

## Deliverables (per assignment)
- Clean Go architecture, stealth/human-behavior layer, state persistence.
- `.env` template, config file, README instructions, demo video link (later).

## Notes
- All browser actions will go through a stealth layer (mouse/typing/timing/fingerprint masking).
- Anti-detection quality is the top priority; this is a PoC, not a production bot.