package saga

import (
	"testing"

	"homework/internal/model"
	"homework/internal/service"
)

func TestSagaOrchestrator_SuccessfulOrder(t *testing.T) {
	orchestrator := createTestOrchestrator()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
		{ProductID: "product2", Quantity: 1, Price: 200.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-1", "order-1", "user1", items)
	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	if result.Execution.Status != SagaStatusCompleted {
		t.Errorf("Expected status %s, got %s", SagaStatusCompleted, result.Execution.Status)
	}

	if result.Execution.OrderID == "" {
		t.Error("Expected OrderID to be set")
	}

	expectedSteps := 5
	if len(result.Execution.Steps) != expectedSteps {
		t.Errorf("Expected %d steps, got %d", expectedSteps, len(result.Execution.Steps))
	}

	for _, step := range result.Execution.Steps {
		if step.Status != StepStatusCompleted {
			t.Errorf("Step %s should be completed, got %s", step.Name, step.Status)
		}
	}

	order, err := orchestrator.GetOrder(result.Execution.OrderID)
	if err != nil {
		t.Fatalf("Expected to get order, got error: %v", err)
	}

	if order.Status != model.OrderStatusConfirmed {
		t.Errorf("Expected order status %s, got %s", model.OrderStatusConfirmed, order.Status)
	}
}

func TestSagaOrchestrator_InventoryFailure(t *testing.T) {
	orchestrator := createTestOrchestrator()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1000, Price: 100.0}, 
	}

	result := orchestrator.ExecuteOrderSaga("saga-2", "order-2", "user1", items)

	if result.Success {
		t.Error("Expected failure because of inventory")
	}

	if result.Execution.Status != SagaStatusCompensated && result.Execution.Status != SagaStatusFailed {
		t.Errorf("Expected status %s or %s, got %s", SagaStatusFailed, SagaStatusCompensated, result.Execution.Status)
	}

	if result.Execution.Status == SagaStatusCompensated {
		t.Log("Compensation executed successfully")
	}

	order, err := orchestrator.GetOrder(result.Execution.OrderID)
	if err == nil {
		if order.Status != model.OrderStatusCancelled && order.Status != model.OrderStatusFailed {
			t.Errorf("Expected order to be cancelled or failed, got %s", order.Status)
		}
	}
}

func TestSagaOrchestrator_PaymentFailure(t *testing.T) {
	billingService := service.NewBillingService()
	billingService.SetShouldFail(true) 
	orchestrator := NewSagaOrchestrator(
		service.NewOrderService(),
		billingService,
		service.NewInventoryService(),
		service.NewDiscountService(),
	)

	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-3", "order-3", "user1", items)

	if result.Success {
		t.Error("Expected failure for payment failure")
	}

	if result.Execution.Status != SagaStatusCompensated {
		t.Errorf("Expected status %s, got %s", SagaStatusCompensated, result.Execution.Status)
	}
}

func TestSagaOrchestrator_OrderWithDiscount(t *testing.T) {
	orchestrator := createTestOrchestrator()
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-4", "order-4", "user1", items)
	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	if result.Execution.Status != SagaStatusCompleted {
		t.Errorf("Expected status %s, got %s", SagaStatusCompleted, result.Execution.Status)
	}

	var discountStep *SagaStep
	for i := range result.Execution.Steps {
		if result.Execution.Steps[i].Name == "apply_discount" {
			discountStep = &result.Execution.Steps[i]
			break
		}
	}

	if discountStep == nil {
		t.Fatal("Expected discount step to be present")
	}

	if discount, ok := discountStep.Result.(*model.Discount); ok && discount != nil {
		if discount.Percentage != 10.0 {
			t.Errorf("Expected discount percentage 10.0, got %.2f", discount.Percentage)
		}

		expectedAmount := 200.0 * 10.0 / 100.0
		if discount.Amount != expectedAmount {
			t.Errorf("Expected discount amount %.2f, got %.2f", expectedAmount, discount.Amount)
		}
	} else {
		t.Error("Expected discount to be applied")
	}
}

func TestSagaOrchestrator_OrderWithoutDiscount(t *testing.T) {
	orderSvc := service.NewOrderService()
	billingSvc := service.NewBillingService()
	inventorySvc := service.NewInventoryService()
	discountSvc := service.NewDiscountService()

	billingSvc.SetUserBalance("user3", 10000.0)
	inventorySvc.SetStock("product1", 100)

	orchestrator := NewSagaOrchestrator(
		orderSvc,
		billingSvc,
		inventorySvc,
		discountSvc,
	)

	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-5", "order-5", "user3", items)

	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	if result.Execution.Status != SagaStatusCompleted {
		t.Errorf("Expected status %s, got %s", SagaStatusCompleted, result.Execution.Status)
	}

	var discountStep *SagaStep
	for i := range result.Execution.Steps {
		if result.Execution.Steps[i].Name == "apply_discount" {
			discountStep = &result.Execution.Steps[i]
			break
		}
	}

	if discountStep == nil {
		t.Fatal("Expected discount step to be present")
	}

	if discount, ok := discountStep.Result.(*model.Discount); ok && discount != nil {
		t.Error("Expected no discount for user without discount")
	}
}

func TestSagaOrchestrator_CompensationOrder(t *testing.T) {
	orderSvc := service.NewOrderService()
	billingService := service.NewBillingService()
	inventorySvc := service.NewInventoryService()
	discountSvc := service.NewDiscountService()

	billingService.SetUserBalance("user1", 10000.0)
	inventorySvc.SetStock("product1", 100)
	discountSvc.SetUserDiscount("user1", 10.0)

	billingService.SetShouldFail(true)

	orchestrator := NewSagaOrchestrator(
		orderSvc,
		billingService,
		inventorySvc,
		discountSvc,
	)

	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-6", "order-6", "user1", items)

	if len(result.Execution.Compensations) == 0 {
		t.Error("Expected compensations to be registered")
	}

	if result.Execution.Status != SagaStatusCompensated {
		t.Errorf("Expected status %s, got %s", SagaStatusCompensated, result.Execution.Status)
	}

	compensationNames := make(map[string]bool)
	for _, comp := range result.Execution.Compensations {
		compensationNames[comp.Name] = true
	}

	if !compensationNames["cancel_order"] {
		t.Error("Expected 'cancel_order' compensation")
	}
	if !compensationNames["release_inventory"] {
		t.Error("Expected 'release_inventory' compensation")
	}

	if !compensationNames["remove_discount"] {
		t.Error("Expected 'remove_discount' compensation for user with discount")
	}

}

func createTestOrchestrator() *SagaOrchestrator {
	orderSvc := service.NewOrderService()
	billingSvc := service.NewBillingService()
	inventorySvc := service.NewInventoryService()
	discountSvc := service.NewDiscountService()

	billingSvc.SetUserBalance("user1", 10000.0)
	inventorySvc.SetStock("product1", 100)
	inventorySvc.SetStock("product2", 100)
	discountSvc.SetUserDiscount("user1", 10.0)

	return NewSagaOrchestrator(
		orderSvc,
		billingSvc,
		inventorySvc,
		discountSvc,
	)
}
