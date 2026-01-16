package service

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"homework/internal/model"
)

type DiscountService struct {
	mu        sync.RWMutex
	discounts map[string]*model.Discount
	userDiscounts map[string]float64
}

func NewDiscountService() *DiscountService {
	service := &DiscountService{
		discounts:     make(map[string]*model.Discount),
		userDiscounts: make(map[string]float64),
	}

	service.userDiscounts["user1"] = 10.0
	service.userDiscounts["user2"] = 15.0

	return service
}

func (s *DiscountService) SetUserDiscount(userID string, percentage float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userDiscounts[userID] = percentage
}

func (s *DiscountService) ApplyDiscount(orderID, userID string, totalAmount float64) (*model.Discount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	discountPercentage, hasDiscount := s.userDiscounts[userID]
	if !hasDiscount {
		return nil, nil
	}

	discountAmount := totalAmount * discountPercentage / 100.0

	discount := &model.Discount{
		ID:         uuid.New().String(),
		UserID:     userID,
		OrderID:    orderID,
		Amount:     discountAmount,
		Percentage: discountPercentage,
	}

	s.discounts[discount.ID] = discount
	return discount, nil
}

func (s *DiscountService) RemoveDiscount(discountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.discounts[discountID]
	if !exists {
		return fmt.Errorf("discount not found: %s", discountID)
	}

	delete(s.discounts, discountID)
	return nil
}

func (s *DiscountService) GetDiscount(discountID string) (*model.Discount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	discount, exists := s.discounts[discountID]
	if !exists {
		return nil, fmt.Errorf("discount not found: %s", discountID)
	}

	return discount, nil
}
