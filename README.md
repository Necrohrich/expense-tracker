# Expense Tracker API

## How to Run
<!-- TODO Saturday -->

## Tech Choices
- Go 1.26, stdlib net/http (1.22+ method+path routing) — no framework, API surface too small to justify Gin/Chi
- modernc.org/sqlite — pure-Go driver, no CGO, keeps "single command to run" true on any machine
- google/uuid for IDs — chosen over auto-increment despite worse B-Tree insert locality (random 
  vs sequential keys); scale is a single local user, so the tradeoff is irrelevant in practice, 
  UUID reads cleaner as a public identifier
- Separate DTOs (CreateExpenseRequest) instead of reusing the Expense model for input — 
  prevents clients from setting server-controlled fields (id, created_at) via request body 
  (mass assignment protection), and narrows accepted fields to exactly what's needed
- writeJSONError() helper for consistent error responses — used across multiple handlers 
  (POST, PATCH, GET/{id}, DELETE all need the same "400/404 + JSON error body" shape), 
  so extracting it avoids repeating the same 3 lines each time
- requests.http (VS Code REST Client) checked into the repo instead of Postman/Swagger — 
  zero setup for the reviewer (no separate app, no import step), doubles as living API 
  documentation, and lets me demo every endpoint live during the interview by just clicking 
  "Send Request" next to each one
- Empty result set returns `[]` in JSON, not `null` — initialized as `expenses := []Expense{}` 
  instead of `var expenses []Expense` (Go's nil slice serializes to `null`, which most API 
  clients don't expect from a list endpoint)
- DELETE existence check via RowsAffected(), not error handling — unlike SELECT (sql.ErrNoRows), 
  a DELETE on a non-existent id succeeds silently with 0 rows affected; RowsAffected() is the only 
  way to detect "nothing was deleted" and return 404 instead of a false 204

## What I'd Improve With More Time
- Amount stored as float64, not integer cents / decimal — known precision risk for financial data, acceptable for this scope
- GET /expenses is O(n) time and space with no pagination — fine at this scale, would add 
  LIMIT/OFFSET and an index on spent_on for larger datasets (avoids in-memory sort)

## Assumptions
- spent_on stored as string (YYYY-MM-DD), validated at two layers: SQL CHECK(GLOB) catches malformed 
  format at the DB level, application-layer time.Parse (handlers.go, WIP) catches invalid-but-well-
  formatted dates like 2026-13-45

## Debugging Notes
- Spent ~20 min on a false-negative SQL CHECK constraint: SQLite's GLOB operator uses `?` 
  as the single-character wildcard, not `_` (that's LIKE's syntax) — `_` in GLOB is a literal 
  character. The constraint `CHECK(spent_on GLOB '____-__-__')` silently rejected every valid 
  date. Fixed to `'????-??-??'
- Spent ~15 min on spent_on/created_at coming back as full timestamps (e.g. "2026-05-08T00:00:00Z" 
  instead of "2026-05-08") despite being stored as plain strings. Root cause: SQLite type affinity — 
  columns declared as DATE/DATETIME caused the driver to auto-convert to time.Time on read, which 
  database/sql then reformatted as RFC3339. Fixed by declaring both columns as TEXT, matching how 
  they're actually used (plain strings, validated at the application layer).