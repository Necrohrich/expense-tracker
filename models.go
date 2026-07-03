package main

type Expense struct {
	ID	string	`json:"id"`
	Amount	float64	`json:"amount"` // optimize in future to int with divide to 100
	Category string `json:"category"`
	Note     string  `json:"note,omitempty"`
	SpentOn  string  `json:"spent_on"` // change to time.Parse("2006-01-02", spentOn) in future
	CreatedAt string `json:"created_at"`
}
