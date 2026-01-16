package main

import (
	"fmt"
	"homework/internal/model"
	"homework/internal/saga"
	"homework/internal/service"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	orderSvc := service.NewOrderService()
	billingSvc := service.NewBillingService()
	inventorySvc := service.NewInventoryService()
	discountSvc := service.NewDiscountService()

	billingSvc.SetUserBalance("dasha", 1000.0)
	billingSvc.SetUserBalance("nastya", 100.0)
	inventorySvc.SetStock("apple", 100)
	discountSvc.SetUserDiscount("dasha", 15.0)

	sagaOrch := saga.NewSagaOrchestrator(orderSvc, billingSvc, inventorySvc, discountSvc)

	items1 := []model.OrderItem{
		{ProductID: "apple", Quantity: 2, Price: 200.0},
	}

	result1 := sagaOrch.ExecuteOrderSaga("saga-1", "order-1", "dasha", items1)
	if result1.Success {
		fmt.Printf("\n✓ Order completed! Balance: %.2f\n\n", billingSvc.GetUserBalance("dasha"))
	}

	items2 := []model.OrderItem{
		{ProductID: "apple", Quantity: 2, Price: 200.0},
	}

	result2 := sagaOrch.ExecuteOrderSaga("saga-2", "order-2", "nastya", items2)
	if !result2.Success {
		fmt.Printf("\n✗ Order failed (expected): %v\n", result2.Error)
		fmt.Printf("Balance unchanged: %.2f\n", billingSvc.GetUserBalance("nastya"))
		fmt.Printf("Stock unchanged: %d\n\n", inventorySvc.GetStock("apple"))
	}

	inventorySvc.SetStock("apple", 30) 

	numSagas := 100
	var successCount int32
	var failCount int32
	var stockFailures int32
	var balanceFailures int32
	var wg sync.WaitGroup

	for i := 0; i < numSagas; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			names := []string{"dasha", "nastya", "tom"}
			userID := names[index%len(names)]
			orderID := fmt.Sprintf("order_%d", index)
			sagaID := fmt.Sprintf("saga_%d", index)

			balance := 500.0
			if index%3 == 0 {
				balance = 50.0 
			}
			billingSvc.SetUserBalance(userID, balance)
			discountSvc.SetUserDiscount(userID, float64(index%10))

			items := []model.OrderItem{
				{ProductID: "apple", Quantity: 1, Price: 100.0},
			}

			result := sagaOrch.ExecuteOrderSaga(sagaID, orderID, userID, items)
			if result.Success {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
				if result.Error != nil {
					errMsg := result.Error.Error()
					if strings.Contains(errMsg, "insufficient stock") {
						atomic.AddInt32(&stockFailures, 1)
					} else if strings.Contains(errMsg, "insufficient funds") {
						atomic.AddInt32(&balanceFailures, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("\n✓ Completed %d sagas successfully\n", successCount)
	fmt.Printf("✗ Failed %d sagas (expected due to limited stock/balance)\n", failCount)
	fmt.Printf("  - Failed due to stock: %d\n", stockFailures)
	fmt.Printf("  - Failed due to balance: %d\n", balanceFailures)
}
