# Expense Tracker API

## How to Run
<!-- TODO Saturday -->

## Tech Choices
- Go 1.26, stdlib net/http (1.22+ method+path routing) — no framework, API surface too small to justify Gin/Chi
- modernc.org/sqlite — pure-Go driver, no CGO, keeps "single command to run" true on any machine
- google/uuid for IDs — chosen over auto-increment for [TODO: your reasoning]

## What I'd Improve With More Time
- Amount stored as float64, not integer cents / decimal — known precision risk for financial data, acceptable for this scope

## Assumptions
- spent_on stored as string (YYYY-MM-DD), no strict format validation — 
  documented trade-off for time budget; would add regex/time.Parse check with more time