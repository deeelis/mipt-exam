package model

type Discount struct {
	ID         string
	UserID     string
	OrderID    string
	Amount     float64
	Percentage float64
}
