package model

import "time"

type Payment struct {
	ID        string
	OrderID   string
	UserID    string
	Amount    float64
	Status    PaymentStatus
	CreatedAt time.Time
}

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)
