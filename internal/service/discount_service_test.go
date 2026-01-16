package service

import (
	"testing"
)

func TestDiscountService_ApplyDiscount(t *testing.T) {
	service := NewDiscountService()
	totalAmount := 200.0

	discount, err := service.ApplyDiscount("order1", "user1", totalAmount)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if discount == nil {
		t.Fatal("Expected discount to be applied")
	}

	if discount.Percentage != 10.0 {
		t.Errorf("Expected discount percentage 10.0, got %.2f", discount.Percentage)
	}

	expectedAmount := 200.0 * 10.0 / 100.0
	if discount.Amount != expectedAmount {
		t.Errorf("Expected discount amount %.2f, got %.2f", expectedAmount, discount.Amount)
	}
}

func TestDiscountService_ApplyDiscount_NoDiscount(t *testing.T) {
	service := NewDiscountService()
	totalAmount := 200.0

	discount, err := service.ApplyDiscount("order1", "user3", totalAmount)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if discount != nil {
		t.Error("Expected no discount for user without discount")
	}
}

func TestDiscountService_RemoveDiscount(t *testing.T) {
	service := NewDiscountService()
	discount, _ := service.ApplyDiscount("order1", "user1", 200.0)

	err := service.RemoveDiscount(discount.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	_, err = service.GetDiscount(discount.ID)
	if err == nil {
		t.Error("Expected discount to be removed")
	}
}
