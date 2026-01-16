package service

import (
	"testing"

	"homework/internal/model"
)

func TestOrderService_CreateOrder(t *testing.T) {
	service := NewOrderService()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
		{ProductID: "product2", Quantity: 1, Price: 200.0},
	}

	order, err := service.CreateOrder("user1", items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if order.ID == "" {
		t.Error("Expected order ID to be set")
	}

	if order.UserID != "user1" {
		t.Errorf("Expected user ID 'user1', got '%s'", order.UserID)
	}

	if order.Status != model.OrderStatusPending {
		t.Errorf("Expected status %s, got %s", model.OrderStatusPending, order.Status)
	}

	expectedTotal := 2*100.0 + 1*200.0
	if order.Total != expectedTotal {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, order.Total)
	}
}

func TestOrderService_ConfirmOrder(t *testing.T) {
	service := NewOrderService()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}
	order, _ := service.CreateOrder("user1", items)

	err := service.ConfirmOrder(order.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	confirmedOrder, _ := service.GetOrder(order.ID)
	if confirmedOrder.Status != model.OrderStatusConfirmed {
		t.Errorf("Expected status %s, got %s", model.OrderStatusConfirmed, confirmedOrder.Status)
	}
}

func TestOrderService_CancelOrder(t *testing.T) {
	service := NewOrderService()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}
	order, _ := service.CreateOrder("user1", items)

	err := service.CancelOrder(order.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	cancelledOrder, _ := service.GetOrder(order.ID)
	if cancelledOrder.Status != model.OrderStatusCancelled {
		t.Errorf("Expected status %s, got %s", model.OrderStatusCancelled, cancelledOrder.Status)
	}
}
