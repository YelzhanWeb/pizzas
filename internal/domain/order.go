package domain

import (
	"errors"
	"time"
)

// Order represents a restaurant order entity
type Order struct {
	ID              int
	Number          string
	CustomerName    string
	Type            OrderType
	TableNumber     *int
	DeliveryAddress *string
	Items           []OrderItem
	TotalAmount     float64
	Priority        Priority
	Status          Status
	ProcessedBy     *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID       int
	OrderID  int
	Name     string
	Quantity int
	Price    float64
}

// NewOrder creates a new order with business rules applied
func NewOrder(customerName string, orderType OrderType, items []OrderItem, tableNumber *int, deliveryAddress *string) (*Order, error) {
	order := &Order{
		CustomerName:    customerName,
		Type:            orderType,
		Items:           items,
		TableNumber:     tableNumber,
		DeliveryAddress: deliveryAddress,
		Status:          StatusReceived,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := order.Validate(); err != nil {
		return nil, err
	}

	order.CalculateTotal()
	order.DeterminePriority()

	return order, nil
}

// Validate applies business validation rules
func (o *Order) Validate() error {
	if len(o.CustomerName) < 1 || len(o.CustomerName) > 100 {
		return errors.New("customer name must be 1-100 characters")
	}

	if o.Type != OrderTypeDineIn && o.Type != OrderTypeTakeout && o.Type != OrderTypeDelivery {
		return errors.New("invalid order type")
	}

	if o.Type == OrderTypeDineIn && o.TableNumber == nil {
		return errors.New("table number required for dine-in orders")
	}

	if o.Type == OrderTypeDineIn && (o.TableNumber != nil && (*o.TableNumber < 1 || *o.TableNumber > 100)) {
		return errors.New("table number must be between 1 and 100")
	}

	if o.Type == OrderTypeDelivery && (o.DeliveryAddress == nil || len(*o.DeliveryAddress) < 10) {
		return errors.New("delivery address required (min 10 characters)")
	}

	if len(o.Items) < 1 || len(o.Items) > 20 {
		return errors.New("order must have 1-20 items")
	}

	for _, item := range o.Items {
		if len(item.Name) < 1 || len(item.Name) > 50 {
			return errors.New("item name must be 1-50 characters")
		}
		if item.Quantity < 1 || item.Quantity > 10 {
			return errors.New("item quantity must be 1-10")
		}
		if item.Price < 0.01 || item.Price > 999.99 {
			return errors.New("item price must be 0.01-999.99")
		}
	}

	return nil
}

// CalculateTotal calculates the total amount of the order
func (o *Order) CalculateTotal() {
	total := 0.0
	for _, item := range o.Items {
		total += item.Price * float64(item.Quantity)
	}
	o.TotalAmount = total
}

// DeterminePriority determines the priority based on total amount
func (o *Order) DeterminePriority() {
	if o.TotalAmount > 100 {
		o.Priority = PriorityHigh
	} else if o.TotalAmount >= 50 {
		o.Priority = PriorityMedium
	} else {
		o.Priority = PriorityLow
	}
}

// TransitionTo transitions the order to a new status
func (o *Order) TransitionTo(newStatus Status, processedBy string) error {
	if !o.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}

	o.Status = newStatus
	o.UpdatedAt = time.Now()

	if processedBy != "" {
		o.ProcessedBy = &processedBy
	}

	if newStatus == StatusReady {
		now := time.Now()
		o.CompletedAt = &now
	}

	return nil
}

// CanTransitionTo checks if the order can transition to the new status
func (o *Order) CanTransitionTo(newStatus Status) bool {
	validTransitions := map[Status][]Status{
		StatusReceived:  {StatusCooking, StatusCancelled},
		StatusCooking:   {StatusReady, StatusCancelled},
		StatusReady:     {StatusCompleted, StatusCancelled},
		StatusCompleted: {},
		StatusCancelled: {},
	}

	allowed := validTransitions[o.Status]
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// GetCookingTime returns the cooking time based on order type
func (o *Order) GetCookingTime() time.Duration {
	switch o.Type {
	case OrderTypeDineIn:
		return 8 * time.Second
	case OrderTypeTakeout:
		return 10 * time.Second
	case OrderTypeDelivery:
		return 12 * time.Second
	default:
		return 10 * time.Second
	}
}

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidOrderType        = errors.New("invalid order type")
)
