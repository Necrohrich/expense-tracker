# Expense Tracker API

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

## Tech Choices
- Go 1.26, stdlib net/http (1.22+ method+path routing) — no framework, API surface too small to justify Gin/Chi
- modernc.org/sqlite — pure-Go driver, no CGO, keeps "single command to run" true on any machine
- google/uuid for IDs — chosen over auto-increment despite worse B-Tree insert locality (random vs sequential keys); scale is a single local user, so the tradeoff is irrelevant in practice, UUID reads cleaner as a public identifier
- Separate DTOs (CreateExpenseRequest) instead of reusing the Expense model for input — prevents clients from setting server-controlled fields (id, created_at) via request body (mass assignment protection), and narrows accepted fields to exactly what's needed
- writeJSONError() helper for consistent error responses — used across multiple handlers (POST, PATCH, GET/{id}, DELETE all need the same "400/404 + JSON error body" shape), so extracting it avoids repeating the same 3 lines each time
- requests.http (VS Code REST Client) checked into the repo instead of Postman/Swagger — zero setup for the reviewer (no separate app, no import step), doubles as living API documentation, and lets me demo every endpoint live during the interview by just clicking "Send Request" next to each one
- Empty result set returns `[]` in JSON, not `null` — initialized as `expenses := []Expense{}` instead of `var expenses []Expense` (Go's nil slice serializes to `null`, which most API clients don't expect from a list endpoint)
- DELETE existence check via RowsAffected(), not error handling — unlike SELECT (sql.ErrNoRows), a DELETE on a non-existent id succeeds silently with 0 rows affected; RowsAffected() is the only way to detect "nothing was deleted" and return 404 instead of a false 204
- PATCH uses SQLite's RETURNING clause (supported since SQLite 3.35, 2021) instead of a separate UPDATE + SELECT — one round-trip instead of two, and reuses the same QueryRow+Scan+errors.Is(sql.ErrNoRows) pattern already used for GET/{id}, so a missing id naturally falls out as 404 without needing RowsAffected() here
- UpdateExpenseRequest fields are pointers (*float64, *string), not plain values — required to distinguish "client didn't send this field" (nil) from "client explicitly sent an empty value" (non-nil pointing to zero value); a plain float64/string can't make that distinction
- Relied on Go 1.22+ ServeMux's built-in method+path pattern matching to disambiguate GET /expenses/{id} from GET /expenses/summary automatically (literal segments take precedence over wildcards), avoiding the need for route ordering tricks common in older routers
- Multi-stage Dockerfile: golang:1.26-alpine builder stage compiles the binary, final stage is bare alpine:latest with only the compiled binary copied in — no Go toolchain or source code in the runtime image, minimizing size and attack surface
- go.mod/go.sum copied and go mod download run before copying the rest of the source, so dependency downloads stay cached across builds when only application code changes
- PORT and DB_PATH configurable via environment variables (os.Getenv), falling back to sensible defaults (8080, expenses.db) when unset — no config file/library needed for just two values
- GET /expenses supports ?page=&limit= query params, defaulting to page=1/limit=20 when absent or invalid — invalid values fall back silently rather than erroring, since pagination affects display only, not data integrity (unlike amount validation on writes)
- Logging via middleware wrapping the whole mux (Decorator pattern), not per-handler calls — avoids repeating logging code in all 6 handlers. Captures status code by wrapping http.ResponseWriter in a statusRecorder that intercepts WriteHeader(), since the interface itself exposes no way to read back what was already written

## What I'd Improve With More Time
- Only one test written (amount validation on POST /expenses) — chosen because it's the one validation rule the spec explicitly calls out as mandatory ("do not skip"). With more time I'd add tests for: 404 handling on GET/PATCH/DELETE with non-existent id, the RETURNING-based PATCH update itself, and the GROUP BY logic in /summary
- Amount stored as float64, not integer cents / decimal — known precision risk for financial data, acceptable for this scope
- No index on spent_on — ORDER BY currently requires an in-memory sort; fine at this scale, would add an index for larger datasets
- Pagination doesn't return total count/page metadata (e.g. total_pages) — client can't tell if there are more pages without requesting page+1 and checking for an empty result

## Assumptions
- spent_on stored as string (YYYY-MM-DD), validated at two layers: SQL CHECK(GLOB) catches malformed format at the DB level, application-layer time.Parse (handlers.go, WIP) catches invalid-but-well-formatted dates like 2026-13-45

## Debugging Notes
- Spent ~20 min on a false-negative SQL CHECK constraint: SQLite's GLOB operator uses `?` as the single-character wildcard, not `_` (that's LIKE's syntax) — `_` in GLOB is a literal character. The constraint `CHECK(spent_on GLOB '____-__-__')` silently rejected every valid date. Fixed to `'????-??-??'
- Spent ~15 min on spent_on/created_at coming back as full timestamps (e.g. "2026-05-08T00:00:00Z" instead of "2026-05-08") despite being stored as plain strings. Root cause: SQLite type affinity — columns declared as DATE/DATETIME caused the driver to auto-convert to time.Time on read, which database/sql then reformatted as RFC3339. Fixed by declaring both columns as TEXT, matching how they're actually used (plain strings, validated at the application layer).