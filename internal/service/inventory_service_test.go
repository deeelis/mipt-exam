package service

import (
	"testing"

	"homework/internal/model"
)

func TestInventoryService_ReserveItems(t *testing.T) {
	service := NewInventoryService()
	service.SetStock("product1", 10)
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
	}

	reservations, err := service.ReserveItems("order1", items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(reservations) != 1 {
		t.Errorf("Expected 1 reservation, got %d", len(reservations))
	}

	product, _ := service.GetProduct("product1")
	if product.Stock != 8 {
		t.Errorf("Expected stock 8, got %d", product.Stock)
	}
}

func TestInventoryService_ReserveItems_InsufficientStock(t *testing.T) {
	service := NewInventoryService()
	service.SetStock("product1", 10)
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1000, Price: 100.0},
	}

	_, err := service.ReserveItems("order1", items)
	if err == nil {
		t.Error("Expected error for insufficient stock")
	}
}

func TestInventoryService_ReleaseItems(t *testing.T) {
	service := NewInventoryService()
	service.SetStock("product1", 10)
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
	}
	service.ReserveItems("order1", items)

	err := service.ReleaseItems("order1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	product, _ := service.GetProduct("product1")
	if product.Stock != 10 {
		t.Errorf("Expected stock 10, got %d", product.Stock)
	}
}
