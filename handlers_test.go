package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestCreateExpense_InvalidAmount(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	body := CreateExpenseRequest{Amount: -5, Category: "food", SpentOn: "2026-05-08"}
    bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/expenses", bytes.NewReader(bodyBytes))
	recorder := httptest.NewRecorder()

	handler := createExpenseHandler(db)
    handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
        t.Errorf("expected status 400, got %d", recorder.Code)
    }
}