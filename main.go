package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "expenses.db"
	}

	port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

	db, err := InitDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /expenses", createExpenseHandler(db))
	mux.HandleFunc("GET /expenses", getExpensesHandler(db))
	mux.HandleFunc("GET /expenses/{id}", getExpenseHandler(db))
	mux.HandleFunc("DELETE /expenses/{id}", deleteExpenseHandler(db))
	mux.HandleFunc("PATCH /expenses/{id}", updateExpenseHandler(db))
	mux.HandleFunc("GET /expenses/summary", getSummaryHandler(db))
	mux.HandleFunc("GET /health", getHealthHandler(db))
	mux.HandleFunc("GET /expenses/count", getExpensesCountHandler(db))
	mux.HandleFunc("GET /expenses/search", getSearchExpensesHandler(db))
	mux.HandleFunc("PATCH /expenses/{id}/soft-delete", softDeleteExpenseHandler(db))

	log.Println("Server starting on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, loggingMiddleware(mux)))
}