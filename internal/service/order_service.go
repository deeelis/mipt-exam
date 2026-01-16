package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"homework/internal/model"
)

type OrderService struct {
	mu     sync.RWMutex
	orders map[string]*model.Order
}

func NewOrderService() *OrderService {
	return &OrderService{
		orders: make(map[string]*model.Order),
	}
}

func (s *OrderService) CreateOrder(userID string, items []model.OrderItem) (*model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order := &model.Order{
		ID:        uuid.New().String(),
		UserID:    userID,
		Items:     items,
		Status:    model.OrderStatusPending,
		CreatedAt: time.Now(),
	}

	for _, item := range items {
		order.Total += item.Price * float64(item.Quantity)
	}

	s.orders[order.ID] = order
	return order, nil
}

func (s *OrderService) ConfirmOrder(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = model.OrderStatusConfirmed
	return nil
}

func (s *OrderService) CancelOrder(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = model.OrderStatusCancelled
	return nil
}

func (s *OrderService) FailOrder(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = model.OrderStatusFailed
	return nil
}

func (s *OrderService) GetOrder(orderID string) (*model.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return order, nil
}
