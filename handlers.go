package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func writeJSONError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func createExpenseHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateExpenseRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return 
		}

		if req.Amount <= 0 {
			writeJSONError(w, http.StatusBadRequest, "amount must be greater than 0")
			return 
		}
		if req.Category == "" {
			writeJSONError(w, http.StatusBadRequest, "category is required")
			return 
		}
		if req.SpentOn == "" {
			writeJSONError(w, http.StatusBadRequest, "spent_on is required")
			return 
		}

		_, err = time.Parse("2006-01-02", req.SpentOn)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "spent_on must be a valid date in YYYY-MM-DD format")
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)

		newExpense := Expense{
			ID:        uuid.New().String(),
			Amount:    req.Amount,
			Category:  req.Category,
			Note:      req.Note,
			SpentOn:   req.SpentOn,
			CreatedAt: now,
		}

		_, err = db.Exec(
			"INSERT INTO expenses (id, amount, category, note, spent_on, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			newExpense.ID, newExpense.Amount, newExpense.Category, newExpense.Note, newExpense.SpentOn, newExpense.CreatedAt,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save expense")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newExpense)
	}
}

func getExpensesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, amount, category, note, spent_on, created_at FROM expenses ORDER BY spent_on DESC")
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch expenses")
            return
		}
		defer rows.Close()

		expenses := []Expense{}  // empty return []
		for rows.Next() {
			var e Expense
			err := rows.Scan(&e.ID, &e.Amount, &e.Category, &e.Note, &e.SpentOn, &e.CreatedAt)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to scan row")
				return
			}
			expenses = append(expenses, e)
		}

		w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(expenses)
	}
}

func getExpenseHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var e Expense
		err := db.QueryRow(
            "SELECT id, amount, category, note, spent_on, created_at FROM expenses WHERE id = ?",
            id,
        ).Scan(&e.ID, &e.Amount, &e.Category, &e.Note, &e.SpentOn, &e.CreatedAt)

		if errors.Is(err, sql.ErrNoRows) { // bes practice for search error, we can use == now but is not preffered
			writeJSONError(w, http.StatusNotFound, "expense not found")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch expense")
			return
		}

		w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(e)
	}
}