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
	// routers

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}