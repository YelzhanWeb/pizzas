package http

import (
	"encoding/json"
	"net/http"

	"github.com/YelzhanWeb/pizzas/internal/adapter/logger"

	"github.com/YelzhanWeb/pizzas/internal/interfaces"
)

type OrderHandler struct {
	service interfaces.OrderService
	logger  logger.Logger
}

func NewOrderHandler(service interfaces.OrderService, logger logger.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cmd := interfaces.CreateOrderCommand{
		CustomerName:    req.CustomerName,
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		Items:           convertItemsToCommand(req.Items),
	}

	result, err := h.service.CreateOrder(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// FIX: Use result.Number instead of result.OrderNumber (Domain model field is Number)
	resp := CreateOrderResponse{
		OrderNumber: result.Number,
		Status:      string(result.Status), // Convert domain.Status to string
		TotalAmount: result.TotalAmount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func convertItemsToCommand(items []OrderItemRequest) []interfaces.CreateOrderItemCommand {
	result := make([]interfaces.CreateOrderItemCommand, len(items))
	for i, item := range items {
		result[i] = interfaces.CreateOrderItemCommand{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}
	return result
}

type CreateOrderRequest struct {
	CustomerName    string             `json:"customer_name"`
	OrderType       string             `json:"order_type"`
	TableNumber     *int               `json:"table_number,omitempty"`
	DeliveryAddress *string            `json:"delivery_address,omitempty"`
	Items           []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type CreateOrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}
