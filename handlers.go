package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

func getHealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.Ping()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database connection failed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	}
}

func getExpensesCountHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var totalCount int64
		err := db.QueryRow("SELECT COUNT(*) FROM expenses WHERE deleted_at IS NULL").Scan(&totalCount)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to count expenses")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"count": totalCount})
	}
}

func getExpensesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limitStr := r.URL.Query().Get("limit")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 20
		}

		offset := (page - 1) * limit

		whereClause := ""
		includeDeleted := r.URL.Query().Get("include_deleted")
		if includeDeleted != "true" {
			whereClause = " WHERE deleted_at IS NULL"
		}

		countQuery := "SELECT COUNT(*) FROM expenses" + whereClause
		var totalCount int64
		err = db.QueryRow(countQuery).Scan(&totalCount)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to count expenses")
			return
		}

		var totalPages int
		if totalCount > 0 {
			totalPages = int((totalCount + int64(limit) - 1) / int64(limit))
		} else {
			totalPages = 0
		}

		selectQuery := "SELECT id, amount, category, note, spent_on, created_at FROM expenses" + 
			whereClause + 
			" ORDER BY spent_on DESC LIMIT ? OFFSET ?"

		rows, err := db.Query(selectQuery, limit, offset)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch expenses")
			return
		}
		defer rows.Close()

		expenses := []Expense{} 
		for rows.Next() {
			var e Expense
			err := rows.Scan(&e.ID, &e.Amount, &e.Category, &e.Note, &e.SpentOn, &e.CreatedAt)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to scan row")
				return
			}
			expenses = append(expenses, e)
		}

		response := ExpensePaginationResponse{
			Expenses:    expenses,
			Limit:       limit,
			Offset:      offset,
			CurrentPage: page,
			TotalPages:  totalPages,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}


func getExpenseHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var e Expense
		err := db.QueryRow(
            "SELECT id, amount, category, note, spent_on, created_at FROM expenses WHERE id = ? AND deleted_at IS NULL",
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

func getSearchExpensesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if query == "" {
			writeJSONError(w, http.StatusBadRequest, "query parameter is required")
			return
		}

		rows, err := db.Query(
			"SELECT id, amount, category, note, spent_on, created_at FROM expenses WHERE (category LIKE ? OR note LIKE ?) AND deleted_at IS NULL ORDER BY spent_on DESC",
			"%"+ strings.ToLower(query) +"%", "%"+ strings.ToLower(query) +"%",
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch expenses")
			return
		}
		defer rows.Close()

		expenses := []Expense{}
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

func deleteExpenseHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")

        result, err := db.Exec("DELETE FROM expenses WHERE id = ?", id)
        if err != nil {
            writeJSONError(w, http.StatusInternalServerError, "failed to delete expense")
            return
        }

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to check delete result")
			return
		}
		// RowsAffected == 0 means no row matched this id — DELETE succeeds silently on SQL level otherwise
		if rowsAffected == 0 {
			writeJSONError(w, http.StatusNotFound, "expense not found")
			return
		}

        w.WriteHeader(http.StatusNoContent) // 204
    }
}

func softDeleteExpenseHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var e Expense
		var deletedAt sql.NullString
		now := time.Now().UTC()

		query := `
			UPDATE expenses 
			SET deleted_at = $1 
			WHERE id = $2 AND deleted_at IS NULL 
			RETURNING id, amount, category, note, spent_on, created_at, deleted_at`

		err := db.QueryRow(query, now, id).Scan(
			&e.ID, &e.Amount, &e.Category, &e.Note, &e.SpentOn, &e.CreatedAt, &deletedAt,
		)

		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "expense not found or already deleted")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to soft-delete expense")
			return
		}

		if deletedAt.Valid {
			e.DeletedAt = &deletedAt.String
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(e)
	}
}



func updateExpenseHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")

        var req UpdateExpenseRequest
        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
            return
        }

        if req.Amount != nil && *req.Amount <= 0 {
            writeJSONError(w, http.StatusBadRequest, "amount must be greater than 0")
            return
        }
        if req.Category != nil && *req.Category == "" {
            writeJSONError(w, http.StatusBadRequest, "category cannot be empty")
            return
        }

        var setClauses []string
        var args []any

        if req.Amount != nil {
            setClauses = append(setClauses, "amount = ?")
            args = append(args, *req.Amount)
        }
        if req.Category != nil {
            setClauses = append(setClauses, "category = ?")
            args = append(args, *req.Category)
        }
        if req.Note != nil {
            setClauses = append(setClauses, "note = ?")
            args = append(args, *req.Note)
        }

        if len(setClauses) == 0 {
            writeJSONError(w, http.StatusBadRequest, "at least one field (amount, category, note) must be provided")
            return
        }

        args = append(args, id)
		// RETURNING avoids a second SELECT round-trip after UPDATE
        query := "UPDATE expenses SET " + strings.Join(setClauses, ", ") +
            " WHERE id = ? RETURNING id, amount, category, note, spent_on, created_at"

        var e Expense
        err = db.QueryRow(query, args...).Scan(&e.ID, &e.Amount, &e.Category, &e.Note, &e.SpentOn, &e.CreatedAt)
        if errors.Is(err, sql.ErrNoRows) {
            writeJSONError(w, http.StatusNotFound, "expense not found")
            return
        }
        if err != nil {
            writeJSONError(w, http.StatusInternalServerError, "failed to update expense")
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(e)
    }
}

func getSummaryHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT category, SUM(amount) FROM expenses GROUP BY category")
		if err != nil {
            writeJSONError(w, http.StatusInternalServerError, "failed to fetch summary")
            return
        }
		defer rows.Close()

		summary := map[string]float64{}
		for rows.Next() {
			var category string
			var total float64
			err := rows.Scan(&category, &total)
			if err != nil{
				writeJSONError(w, http.StatusInternalServerError, "failed to scan row")
                return
			}
			summary[category] = total
		}
		
		w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(summary)
	}
}