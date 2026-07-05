package main

type CreateExpenseRequest struct {
    Amount   float64 `json:"amount"`
    Category string  `json:"category"`
    Note     string  `json:"note,omitempty"`
    SpentOn  string  `json:"spent_on"`
}

// pointer allows distinguishing "field omitted" (nil) from "field explicitly cleared" (empty string)
type UpdateExpenseRequest struct {
    Amount   *float64 `json:"amount,omitempty"`
    Category *string  `json:"category,omitempty"`
    Note     *string  `json:"note,omitempty"`
}