# Expense Tracker API

A REST API for tracking personal expenses, built in Go using only the standard library.

## How to Run
1. `go mod download`
2. `go run .` — starts on :8080 (or PORT env var)
3. `go test ./...` — runs tests

Environment variables (optional):
- `PORT` — server port (default: 8080)
- `DB_PATH` — SQLite file path (default: expenses.db)

Docker:
- `docker build -t expense-tracker .`
- `docker run -p 8080:8080 -v ${PWD}/expenses.db:/app/expenses.db expense-tracker`

**Testing manually:** `requests.http` (VS Code REST Client) contains ready requests for all 6 endpoints. IDs are hardcoded from prior runs — after a fresh start, run the POST request first, copy the returned `id`, and use it in the GET/PATCH/DELETE requests below.

## Tech Choices
- Go 1.26, stdlib net/http (1.22+ method+path routing) — no framework, API surface too small to justify Gin/Chi
- modernc.org/sqlite — pure-Go driver, no CGO, keeps "single command to run" true on any machine
- google/uuid for IDs — chosen over auto-increment despite worse B-Tree insert locality (random vs sequential keys); scale is a single local user, so the tradeoff is irrelevant in practice, UUID reads cleaner as a public identifier
- PATCH uses SQLite's RETURNING clause (available since SQLite 3.35, 2021) to update and re-fetch the row in one round-trip, instead of separate UPDATE + SELECT queries
- UpdateExpenseRequest uses pointer fields (*float64, *string) to distinguish "field omitted" (nil) from "field explicitly cleared" (non-nil, zero value) — required for true partial updates
- DELETE checks RowsAffected() for existence, since a DELETE on a non-existent id succeeds silently in SQL (0 rows affected) rather than returning an error
- `[]Expense{}` initialization ensures GET /expenses returns `[]` (not `null`) for an empty result set — Go's nil slice serializes to `null`, which most API clients don't expect

## What I'd Improve With More Time
- Only one test written (amount validation on POST /expenses) — chosen because it's the one validation rule the spec explicitly calls out as mandatory ("do not skip"). With more time I'd add tests for: 404 handling on GET/PATCH/DELETE with non-existent id, the RETURNING-based PATCH update itself, and the GROUP BY logic in /summary
- Amount stored as float64, not integer cents / decimal — known precision risk for financial data, acceptable for this scope
- No index on spent_on — ORDER BY currently requires an in-memory sort; fine at this scale, would add an index for larger datasets
- Pagination doesn't return total count/page metadata (e.g. total_pages) — client can't tell if there are more pages without requesting page+1 and checking for an empty result

## Assumptions
- spent_on stored as string (YYYY-MM-DD), validated at two layers: SQL CHECK(GLOB) catches malformed format at the DB level, application-layer time.Parse (handlers.go, WIP) catches invalid-but-well-formatted dates like 2026-13-45

## Ambiguity Handled
The spec doesn't state whether `spent_on` can be changed via PATCH — the endpoint 
description lists only "amount, category, or note". Decision: `spent_on` is treated as 
immutable after creation (not accepted by `UpdateExpenseRequest`), since a purchase date is a historical fact about the transaction, not something that should change after the fact. If editable, it would need the same two-layer validation (GLOB + time.Parse) already used on creation.

## Debugging Notes
- Spent ~20 min on a false-negative SQL CHECK constraint: SQLite's GLOB operator uses `?` as the single-character wildcard, not `_` (that's LIKE's syntax) — `_` in GLOB is a literal character. The constraint `CHECK(spent_on GLOB '____-__-__')` silently rejected every valid date. Fixed to `'????-??-??'
- Spent ~15 min on spent_on/created_at coming back as full timestamps (e.g. "2026-05-08T00:00:00Z" instead of "2026-05-08") despite being stored as plain strings. Root cause: SQLite type affinity — columns declared as DATE/DATETIME caused the driver to auto-convert to time.Time on read, which database/sql then reformatted as RFC3339. Fixed by declaring both columns as TEXT, matching how they're actually used (plain strings, validated at the application layer).