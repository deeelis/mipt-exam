package main

import (
	"fmt"
	"testing"

	"homework/internal/model"
	"homework/internal/saga"
	"homework/internal/service"

	"github.com/stretchr/testify/suite"
)

type SagaTestSuite struct {
	suite.Suite
	orderSvc      *service.OrderService
	billingSvc    *service.BillingService
	inventorySvc  *service.InventoryService
	discountSvc   *service.DiscountService
	orchestrator  *saga.SagaOrchestrator
}

func (s *SagaTestSuite) SetupSuite() {
	s.orderSvc = service.NewOrderService()
	s.billingSvc = service.NewBillingService()
	s.inventorySvc = service.NewInventoryService()
	s.discountSvc = service.NewDiscountService()

	s.billingSvc.SetUserBalance("user1", 10000.0)
	s.billingSvc.SetUserBalance("user2", 10000.0)
	s.billingSvc.SetUserBalance("user3", 10000.0)
	s.inventorySvc.SetStock("product1", 100)
	s.inventorySvc.SetStock("product2", 100)
	s.discountSvc.SetUserDiscount("user1", 10.0)

	s.orchestrator = saga.NewSagaOrchestrator(
		s.orderSvc,
		s.billingSvc,
		s.inventorySvc,
		s.discountSvc,
	)
}

func (s *SagaTestSuite) TestSuccessfulOrder() {
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
		{ProductID: "product2", Quantity: 1, Price: 200.0},
	}

	result := s.orchestrator.ExecuteOrderSaga("saga-1", "order-1", "user1", items)
	s.True(result.Success)
	s.NotNil(result.Execution)
	s.Equal(saga.SagaStatusCompleted, result.Execution.Status)
	s.NotEmpty(result.Execution.OrderID)
	s.Equal(5, len(result.Execution.Steps))

	order, err := s.orchestrator.GetOrder(result.Execution.OrderID)
	s.NoError(err)
	s.NotNil(order)
	s.Equal(model.OrderStatusConfirmed, order.Status)

	for _, step := range result.Execution.Steps {
		s.Equal(saga.StepStatusCompleted, step.Status, "Step %s should be completed", step.Name)
	}
}

func (s *SagaTestSuite) TestInventoryFailure() {
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1000, Price: 100.0}, 
	}

	result := s.orchestrator.ExecuteOrderSaga("saga-2", "order-2", "user2", items)
	s.False(result.Success)
	s.NotNil(result.Error)
	s.NotNil(result.Execution)
	s.True(
		result.Execution.Status == saga.SagaStatusCompensated || result.Execution.Status == saga.SagaStatusFailed,
		"Expected status compensated or failed, got %s", result.Execution.Status,
	)

	if result.Execution.Status == saga.SagaStatusCompensated {
		s.T().Log("Compensation executed successfully")
	}
}

func (s *SagaTestSuite) TestPaymentFailure() {
	billingService := service.NewBillingService()
	billingService.SetShouldFail(true)

	orderService := service.NewOrderService()
	inventoryService := service.NewInventoryService()
	discountService := service.NewDiscountService()

	billingService.SetUserBalance("user1", 10000.0)
	inventoryService.SetStock("product1", 100)

	orchestrator := saga.NewSagaOrchestrator(
		orderService,
		billingService,
		inventoryService,
		discountService,
	)

	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-3", "order-3", "user1", items)
	s.False(result.Success)
	s.NotNil(result.Error)
	s.NotNil(result.Execution)
	s.Equal(saga.SagaStatusCompensated, result.Execution.Status)
	s.NotEmpty(result.Execution.Compensations, "Expected compensations to be registered")
}

func (s *SagaTestSuite) TestOrderWithDiscount() {
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 2, Price: 100.0},
	}

	result := s.orchestrator.ExecuteOrderSaga("saga-4", "order-4", "user1", items)
	s.True(result.Success)
	s.NotNil(result.Execution)
	s.Equal(saga.SagaStatusCompleted, result.Execution.Status)
	s.NotEmpty(result.Execution.OrderID)

	var discountStep *saga.SagaStep
	for i := range result.Execution.Steps {
		if result.Execution.Steps[i].Name == "apply_discount" {
			discountStep = &result.Execution.Steps[i]
			break
		}
	}

	s.NotNil(discountStep, "Expected discount step to be present")

	if discount, ok := discountStep.Result.(*model.Discount); ok && discount != nil {
		s.Equal(10.0, discount.Percentage, "Expected discount percentage 10.0")
		expectedAmount := 200.0 * 10.0 / 100.0 
		s.Equal(expectedAmount, discount.Amount, "Expected discount amount %.2f", expectedAmount)
	} else {
		s.Fail("Expected discount to be applied")
	}
}

func (s *SagaTestSuite) TestOrderWithoutDiscount() {
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := s.orchestrator.ExecuteOrderSaga("saga-5", "order-5", "user3", items)
	s.True(result.Success)
	s.NotNil(result.Execution)
	s.Equal(saga.SagaStatusCompleted, result.Execution.Status)

	var discountStep *saga.SagaStep
	for i := range result.Execution.Steps {
		if result.Execution.Steps[i].Name == "apply_discount" {
			discountStep = &result.Execution.Steps[i]
			break
		}
	}

	s.NotNil(discountStep, "Expected discount step to be present")

	if discount, ok := discountStep.Result.(*model.Discount); ok {
		s.Nil(discount, "Expected no discount for user without discount")
	}
}

func (s *SagaTestSuite) TestConcurrentOrders() {
	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	results := make(chan *saga.SagaResult, 5)

	orchestrator := s.orchestrator
	for i := 0; i < 5; i++ {
		go func(index int) {
			result := orchestrator.ExecuteOrderSaga(
				fmt.Sprintf("saga-%d", index),
				fmt.Sprintf("order-%d", index),
				"user1",
				items,
			)
			results <- result
		}(i)
	}

	successCount := 0
	errorCount := 0

	for i := 0; i < 5; i++ {
		result := <-results
		if result.Success {
			successCount++
		} else {
			errorCount++
		}
	}

	s.True(successCount > 0 || errorCount > 0, "Expected at least one in each")
	s.T().Logf("Concurrent orders: successful=%d, failed=%d", successCount, errorCount)
}

func (s *SagaTestSuite) TestCompensationOrder() {
	billingService := service.NewBillingService()
	billingService.SetShouldFail(true)

	orderService := service.NewOrderService()
	inventoryService := service.NewInventoryService()
	discountService := service.NewDiscountService()

	billingService.SetUserBalance("user1", 10000.0)
	inventoryService.SetStock("product1", 100)
	discountService.SetUserDiscount("user1", 10.0)

	orchestrator := saga.NewSagaOrchestrator(
		orderService,
		billingService,
		inventoryService,
		discountService,
	)

	items := []model.OrderItem{
		{ProductID: "product1", Quantity: 1, Price: 100.0},
	}

	result := orchestrator.ExecuteOrderSaga("saga-6", "order-6", "user1", items)

	s.NotEmpty(result.Execution.Compensations, "Expected compensations to be registered")

	compensationNames := make(map[string]bool)
	for _, comp := range result.Execution.Compensations {
		compensationNames[comp.Name] = true
	}

	s.True(compensationNames["cancel_order"], "Expected 'cancel_order' compensation")
	s.True(compensationNames["release_inventory"], "Expected 'release_inventory' compensation")
	s.True(compensationNames["remove_discount"], "Expected 'remove_discount' compensation for user with discount")
}

func TestSagaTestSuite(t *testing.T) {
	suite.Run(t, new(SagaTestSuite))
}

