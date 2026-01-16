package saga

import (
	"fmt"
	"sync"
	"time"

	"homework/internal/model"
	"homework/internal/service"
)

type SagaOrchestrator struct {
	orderService      *service.OrderService
	billingService    *service.BillingService
	inventoryService  *service.InventoryService
	discountService   *service.DiscountService

	mu     sync.RWMutex
	sagas  map[string]*SagaExecution
}

type SagaExecution struct {
	ID            string
	OrderID       string
	UserID        string
	Status        SagaStatus
	Steps         []SagaStep
	Compensations []CompensationAction
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SagaStatus string

const (
	SagaStatusInProgress SagaStatus = "in_progress"
	SagaStatusCompleted  SagaStatus = "completed"
	SagaStatusFailed     SagaStatus = "failed"
	SagaStatusCompensated SagaStatus = "compensated"
)

type SagaStep struct {
	Name      string
	Status    StepStatus
	Error     error
	Result    interface{}
}

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
)

type CompensationAction struct {
	Name   string
	Action func() error
}

type SagaResult struct {
	Success bool
	Error   error
	Execution *SagaExecution
}

func NewSagaOrchestrator(
	orderService *service.OrderService,
	billingService *service.BillingService,
	inventoryService *service.InventoryService,
	discountService *service.DiscountService,
) *SagaOrchestrator {
	return &SagaOrchestrator{
		orderService:     orderService,
		billingService:   billingService,
		inventoryService: inventoryService,
		discountService:  discountService,
		sagas:            make(map[string]*SagaExecution),
	}
}

func (o *SagaOrchestrator) ExecuteOrderSaga(sagaID, orderID, userID string, items []model.OrderItem) *SagaResult {
	now := time.Now()

	execution := &SagaExecution{
		ID:            sagaID,
		OrderID:       orderID,
		UserID:        userID,
		Status:        SagaStatusInProgress,
		Steps:         make([]SagaStep, 0),
		Compensations: make([]CompensationAction, 0),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	o.mu.Lock()
	o.sagas[sagaID] = execution
	o.mu.Unlock()

	_, err := o.executeSaga(execution, userID, items, false)

	result := &SagaResult{
		Success:   err == nil && execution.Status == SagaStatusCompleted,
		Error:     err,
		Execution: execution,
	}

	return result
}


func (o *SagaOrchestrator) executeSaga(execution *SagaExecution, userID string, items []model.OrderItem, async bool) (*SagaExecution, error) {
	step1 := SagaStep{Name: "create_order", Status: StepStatusPending}
	o.updateExecution(execution)
	if async {
		time.Sleep(100 * time.Millisecond)
	}

	order, err := o.orderService.CreateOrder(userID, items)
	if err != nil {
		step1.Status = StepStatusFailed
		step1.Error = err
		execution.Steps = append(execution.Steps, step1)
		execution.Status = SagaStatusFailed
		o.updateExecution(execution)
		return execution, err
	}
	step1.Status = StepStatusCompleted
	step1.Result = order
	execution.OrderID = order.ID
	execution.Steps = append(execution.Steps, step1)
	o.updateExecution(execution)

	execution.Compensations = append(execution.Compensations, CompensationAction{
		Name: "cancel_order",
		Action: func() error {
			return o.orderService.CancelOrder(order.ID)
		},
	})

	step2 := SagaStep{Name: "reserve_inventory", Status: StepStatusPending}
	o.updateExecution(execution)
	if async {
		time.Sleep(150 * time.Millisecond)
	}

	reservations, err := o.inventoryService.ReserveItems(order.ID, items)
	if err != nil {
		step2.Status = StepStatusFailed
		step2.Error = err
		execution.Steps = append(execution.Steps, step2)
		execution.Status = SagaStatusFailed
		o.updateExecution(execution)
		o.compensate(execution)
		return execution, err
	}
	step2.Status = StepStatusCompleted
	step2.Result = reservations
	execution.Steps = append(execution.Steps, step2)
	o.updateExecution(execution)

	execution.Compensations = append(execution.Compensations, CompensationAction{
		Name: "release_inventory",
		Action: func() error {
			return o.inventoryService.ReleaseItems(order.ID)
		},
	})

	step3 := SagaStep{Name: "apply_discount", Status: StepStatusPending}
	o.updateExecution(execution)
	if async {
		time.Sleep(100 * time.Millisecond)
	}

	discount, err := o.discountService.ApplyDiscount(order.ID, userID, order.Total)
	if err != nil {
		step3.Status = StepStatusFailed
		step3.Error = err
		execution.Steps = append(execution.Steps, step3)
		execution.Status = SagaStatusFailed
		o.updateExecution(execution)
		o.compensate(execution)
		return execution, err
	}
	step3.Status = StepStatusCompleted
	step3.Result = discount
	execution.Steps = append(execution.Steps, step3)
	o.updateExecution(execution)

	finalAmount := order.Total
	if discount != nil {
		finalAmount -= discount.Amount
	}

	if discount != nil {
		execution.Compensations = append(execution.Compensations, CompensationAction{
			Name: "remove_discount",
			Action: func() error {
				return o.discountService.RemoveDiscount(discount.ID)
			},
		})
	}

	step4 := SagaStep{Name: "process_payment", Status: StepStatusPending}
	o.updateExecution(execution)
	if async {
		time.Sleep(200 * time.Millisecond)
	}

	payment, err := o.billingService.ProcessPayment(order.ID, userID, finalAmount)
	if err != nil {
		step4.Status = StepStatusFailed
		step4.Error = err
		execution.Steps = append(execution.Steps, step4)
		execution.Status = SagaStatusFailed
		o.updateExecution(execution)
		o.compensate(execution)
		return execution, err
	}
	step4.Status = StepStatusCompleted
	step4.Result = payment
	execution.Steps = append(execution.Steps, step4)
	o.updateExecution(execution)

	execution.Compensations = append(execution.Compensations, CompensationAction{
		Name: "refund_payment",
		Action: func() error {
			return o.billingService.RefundPaymentByOrderID(order.ID)
		},
	})

	step5 := SagaStep{Name: "confirm_order", Status: StepStatusPending}
	o.updateExecution(execution)
	if async {
		time.Sleep(100 * time.Millisecond)
	}

	err = o.orderService.ConfirmOrder(order.ID)
	if err != nil {
		step5.Status = StepStatusFailed
		step5.Error = err
		execution.Steps = append(execution.Steps, step5)
		execution.Status = SagaStatusFailed
		o.updateExecution(execution)
		o.compensate(execution)
		return execution, err
	}
	step5.Status = StepStatusCompleted
	execution.Steps = append(execution.Steps, step5)
	o.updateExecution(execution)

	execution.Status = SagaStatusCompleted
	o.updateExecution(execution)
	return execution, nil
}

func (o *SagaOrchestrator) updateExecution(execution *SagaExecution) {
	execution.UpdatedAt = time.Now()
	o.mu.Lock()
	o.sagas[execution.ID] = execution
	o.mu.Unlock()
}

func (o *SagaOrchestrator) compensate(execution *SagaExecution) {
	execution.Status = SagaStatusCompensated
	
	for i := len(execution.Compensations) - 1; i >= 0; i-- {
		compensation := execution.Compensations[i]
		if err := compensation.Action(); err != nil {
			fmt.Printf("Compensation failed for %s: %v\n", compensation.Name, err)
		}
	}
}

func (o *SagaOrchestrator) GetSagaExecution(sagaID string) (*SagaExecution, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	execution, exists := o.sagas[sagaID]
	if !exists {
		return nil, fmt.Errorf("saga execution not found: %s", sagaID)
	}

	return execution, nil
}

func (o *SagaOrchestrator) GetOrder(orderID string) (*model.Order, error) {
	return o.orderService.GetOrder(orderID)
}

func (o *SagaOrchestrator) GetAllSagas() []*SagaExecution {
	o.mu.RLock()
	defer o.mu.RUnlock()

	sagas := make([]*SagaExecution, 0, len(o.sagas))
	for _, saga := range o.sagas {
		sagas = append(sagas, saga)
	}

	return sagas
}

func (o *SagaOrchestrator) WaitForSagaCompletion(sagaID string, timeout time.Duration) (*SagaExecution, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		execution, err := o.GetSagaExecution(sagaID)
		if err != nil {
			return nil, err
		}

		if execution.Status == SagaStatusCompleted ||
			execution.Status == SagaStatusFailed ||
			execution.Status == SagaStatusCompensated {
			return execution, nil
		}

		<-ticker.C
	}

	execution, err := o.GetSagaExecution(sagaID)
	if err != nil {
		return nil, err
	}

	return execution, fmt.Errorf("saga did not complete within timeout: status=%s", execution.Status)
}
