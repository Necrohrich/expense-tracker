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

## What I'd Improve With More Time
- Amount stored as float64, not integer cents / decimal — known precision risk for financial data, acceptable for this scope

## Assumptions
- spent_on stored as string (YYYY-MM-DD), validated at two layers: SQL CHECK(GLOB) catches malformed 
  format at the DB level, application-layer time.Parse (handlers.go, WIP) catches invalid-but-well-
  formatted dates like 2026-13-45

## Debugging Notes
- Spent ~20 min on a false-negative SQL CHECK constraint: SQLite's GLOB operator uses `?` 
  as the single-character wildcard, not `_` (that's LIKE's syntax) — `_` in GLOB is a literal 
  character. The constraint `CHECK(spent_on GLOB '____-__-__')` silently rejected every valid 
  date. Fixed to `'????-??-??'