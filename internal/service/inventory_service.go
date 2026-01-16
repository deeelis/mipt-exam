package service

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"homework/internal/model"
)

type InventoryService struct {
	mu            sync.RWMutex
	products      map[string]*model.Product
	reservations  map[string]*model.InventoryReservation
	shouldFail bool
}

func NewInventoryService() *InventoryService {
	service := &InventoryService{
		products:     make(map[string]*model.Product),
		reservations: make(map[string]*model.InventoryReservation),
	}

	return service
}

func (s *InventoryService) SetShouldFail(shouldFail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shouldFail = shouldFail
}

func (s *InventoryService) ReserveItems(orderID string, items []model.OrderItem) ([]*model.InventoryReservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shouldFail {
		return nil, fmt.Errorf("inventory reservation failed: insufficient stock")
	}

	var reservations []*model.InventoryReservation

	for _, item := range items {
		product, exists := s.products[item.ProductID]
		if !exists {
			return nil, fmt.Errorf("product not found: %s", item.ProductID)
		}

		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %s: requested %d, available %d",
				item.ProductID, item.Quantity, product.Stock)
		}

		product.Stock -= item.Quantity

		reservation := &model.InventoryReservation{
			ID:        uuid.New().String(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Status:    model.ReservationStatusReserved,
		}

		s.reservations[reservation.ID] = reservation
		reservations = append(reservations, reservation)
	}

	return reservations, nil
}

func (s *InventoryService) ReleaseItems(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, reservation := range s.reservations {
		if reservation.OrderID == orderID && reservation.Status == model.ReservationStatusReserved {
			product, exists := s.products[reservation.ProductID]
			if exists {
				product.Stock += reservation.Quantity
			}
			reservation.Status = model.ReservationStatusReleased
		}
	}

	return nil
}

func (s *InventoryService) SetStock(productID string, stock int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.products[productID]
	if !exists {
		product = &model.Product{
			ID:    productID,
			Price: 0.0,
			Stock: stock,
		}
		s.products[productID] = product
	} else {
		product.Stock = stock
	}
}

func (s *InventoryService) GetStock(productID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, exists := s.products[productID]
	if !exists {
		return 0
	}

	return product.Stock
}

func (s *InventoryService) GetProduct(productID string) (*model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, exists := s.products[productID]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	return product, nil
}
