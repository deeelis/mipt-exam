package service

import (
	"testing"

	"homework/internal/model"
)

func TestBillingService_ProcessPayment(t *testing.T) {
	service := NewBillingService()
	service.SetUserBalance("user1", 1000.0)

	payment, err := service.ProcessPayment("order1", "user1", 100.0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if payment.ID == "" {
		t.Error("Expected payment ID to be set")
	}

	if payment.Status != model.PaymentStatusCompleted {
		t.Errorf("Expected status %s, got %s", model.PaymentStatusCompleted, payment.Status)
	}

	if payment.Amount != 100.0 {
		t.Errorf("Expected amount 100.0, got %.2f", payment.Amount)
	}

	balance := service.GetUserBalance("user1")
	if balance != 900.0 {
		t.Errorf("Expected balance 900.0, got %.2f", balance)
	}
}

func TestBillingService_ProcessPayment_Failure(t *testing.T) {
	service := NewBillingService()
	service.SetShouldFail(true)

	_, err := service.ProcessPayment("order1", "user1", 100.0)
	if err == nil {
		t.Error("Expected error for payment failure")
	}
}

func TestBillingService_RefundPayment(t *testing.T) {
	service := NewBillingService()
	service.SetUserBalance("user1", 1000.0)
	payment, _ := service.ProcessPayment("order1", "user1", 100.0)

	err := service.RefundPayment(payment.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	refundedPayment, _ := service.GetPayment(payment.ID)
	if refundedPayment.Status != model.PaymentStatusRefunded {
		t.Errorf("Expected status %s, got %s", model.PaymentStatusRefunded, refundedPayment.Status)
	}
}

func TestBillingService_RefundPaymentByOrderID(t *testing.T) {
	service := NewBillingService()
	service.SetUserBalance("user1", 1000.0)
	service.ProcessPayment("order1", "user1", 100.0)

	err := service.RefundPaymentByOrderID("order1")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	balance := service.GetUserBalance("user1")
	if balance != 1000.0 {
		t.Errorf("Expected balance 1000.0, got %.2f", balance)
	}
}
