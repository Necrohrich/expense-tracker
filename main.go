package main

import (
	"log"
	"net/http"
)

func main() {
	db, err := InitDB("expenses.db")
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

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}