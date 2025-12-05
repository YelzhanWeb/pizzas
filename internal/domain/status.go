package domain

import "time"

type OrderType string

const (
	OrderTypeDineIn   OrderType = "dine_in"
	OrderTypeTakeout  OrderType = "takeout"
	OrderTypeDelivery OrderType = "delivery"
)

type Status string

const (
	StatusReceived  Status = "received"
	StatusCooking   Status = "cooking"
	StatusReady     Status = "ready"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

// StatusLog represents a log entry for order status changes
type StatusLog struct {
	ID        int
	OrderID   int
	Status    Status
	ChangedBy string
	ChangedAt time.Time
	Notes     *string
}
