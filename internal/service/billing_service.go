package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"homework/internal/model"
)

type BillingService struct {
	mu         sync.RWMutex
	payments   map[string]*model.Payment
	userBalances map[string]float64
	shouldFail bool
}

func NewBillingService() *BillingService {
	return &BillingService{
		payments:     make(map[string]*model.Payment),
		userBalances: make(map[string]float64),
	}
}

func (s *BillingService) SetShouldFail(shouldFail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shouldFail = shouldFail
}

func (s *BillingService) SetUserBalance(userID string, balance float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userBalances[userID] = balance
}

func (s *BillingService) GetUserBalance(userID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userBalances[userID]
}

func (s *BillingService) ProcessPayment(orderID, userID string, amount float64) (*model.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shouldFail {
		return nil, fmt.Errorf("payment processing failed: insufficient funds")
	}

	balance := s.userBalances[userID]
	if balance < amount {
		return nil, fmt.Errorf("insufficient funds: balance %.2f, required %.2f", balance, amount)
	}

	s.userBalances[userID] = balance - amount

	payment := &model.Payment{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		UserID:    userID,
		Amount:    amount,
		Status:    model.PaymentStatusCompleted,
		CreatedAt: time.Now(),
	}

	s.payments[payment.ID] = payment
	return payment, nil
}

func (s *BillingService) RefundPayment(paymentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, exists := s.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found: %s", paymentID)
	}

	payment.Status = model.PaymentStatusRefunded
	return nil
}

func (s *BillingService) RefundPaymentByOrderID(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, payment := range s.payments {
		if payment.OrderID == orderID && payment.Status == model.PaymentStatusCompleted {
			payment.Status = model.PaymentStatusRefunded
			s.userBalances[payment.UserID] += payment.Amount
			return nil
		}
	}

	return fmt.Errorf("payment not found for order: %s", orderID)
}

func (s *BillingService) GetPayment(paymentID string) (*model.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payment, exists := s.payments[paymentID]
	if !exists {
		return nil, fmt.Errorf("payment not found: %s", paymentID)
	}

	return payment, nil
}
