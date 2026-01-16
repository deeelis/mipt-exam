package model

type InventoryReservation struct {
	ID        string
	OrderID   string
	ProductID string
	Quantity  int
	Status    ReservationStatus
}

type ReservationStatus string

const (
	ReservationStatusReserved ReservationStatus = "reserved"
	ReservationStatusReleased  ReservationStatus = "released"
	ReservationStatusFailed    ReservationStatus = "failed"
)
