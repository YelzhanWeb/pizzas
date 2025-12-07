package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error  string            `json:"error"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Регулярное выражение для валидации имени клиента
// Разрешены: буквы, пробелы, дефисы, апострофы
var customerNameRegex = regexp.MustCompile(`^[a-zA-Z\s\-']+$`)

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed, nil)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	// Валидация входных данных
	if validationErrors := validateCreateOrderRequest(req); len(validationErrors) > 0 {
		h.logger.Error("validation_failed", "Order validation failed", "", map[string]interface{}{
			"errors": validationErrors,
		}, fmt.Errorf("validation failed"))

		h.respondError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	cmd := interfaces.CreateOrderCommand{
		CustomerName:    strings.TrimSpace(req.CustomerName),
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		Items:           convertItemsToCommand(req.Items),
	}

	result, err := h.service.CreateOrder(r.Context(), cmd)
	if err != nil {
		h.logger.Error("order_creation_failed", "Failed to create order", "", nil, err)
		h.respondError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	resp := CreateOrderResponse{
		OrderNumber: result.Number,
		Status:      string(result.Status),
		TotalAmount: result.TotalAmount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func validateCreateOrderRequest(req CreateOrderRequest) []ValidationError {
	var errors []ValidationError

	// 1. Валидация customer_name
	customerName := strings.TrimSpace(req.CustomerName)
	if len(customerName) < 1 {
		errors = append(errors, ValidationError{
			Field:   "customer_name",
			Message: "customer name is required",
		})
	} else if len(customerName) > 100 {
		errors = append(errors, ValidationError{
			Field:   "customer_name",
			Message: "customer name must not exceed 100 characters",
		})
	} else if !customerNameRegex.MatchString(customerName) {
		errors = append(errors, ValidationError{
			Field:   "customer_name",
			Message: "customer name must contain only letters, spaces, hyphens, and apostrophes",
		})
	}

	// 2. Валидация order_type
	validOrderTypes := map[string]bool{
		"dine_in":  true,
		"takeout":  true,
		"delivery": true,
	}

	if !validOrderTypes[req.OrderType] {
		errors = append(errors, ValidationError{
			Field:   "order_type",
			Message: "order type must be one of: dine_in, takeout, delivery",
		})
	}

	// 3. Условная валидация в зависимости от order_type
	switch req.OrderType {
	case "dine_in":
		// table_number обязателен
		if req.TableNumber == nil {
			errors = append(errors, ValidationError{
				Field:   "table_number",
				Message: "table number is required for dine-in orders",
			})
		} else if *req.TableNumber < 1 || *req.TableNumber > 100 {
			errors = append(errors, ValidationError{
				Field:   "table_number",
				Message: "table number must be between 1 and 100",
			})
		}

		// delivery_address не должен присутствовать
		if req.DeliveryAddress != nil {
			errors = append(errors, ValidationError{
				Field:   "delivery_address",
				Message: "delivery address must not be present for dine-in orders",
			})
		}

	case "delivery":
		// delivery_address обязателен
		if req.DeliveryAddress == nil {
			errors = append(errors, ValidationError{
				Field:   "delivery_address",
				Message: "delivery address is required for delivery orders",
			})
		} else if len(strings.TrimSpace(*req.DeliveryAddress)) < 10 {
			errors = append(errors, ValidationError{
				Field:   "delivery_address",
				Message: "delivery address must be at least 10 characters",
			})
		}

		// table_number не должен присутствовать
		if req.TableNumber != nil {
			errors = append(errors, ValidationError{
				Field:   "table_number",
				Message: "table number must not be present for delivery orders",
			})
		}

	case "takeout":
		// Ни table_number, ни delivery_address не должны присутствовать
		if req.TableNumber != nil {
			errors = append(errors, ValidationError{
				Field:   "table_number",
				Message: "table number must not be present for takeout orders",
			})
		}
		if req.DeliveryAddress != nil {
			errors = append(errors, ValidationError{
				Field:   "delivery_address",
				Message: "delivery address must not be present for takeout orders",
			})
		}
	}

	// 4. Валидация items
	if len(req.Items) < 1 {
		errors = append(errors, ValidationError{
			Field:   "items",
			Message: "order must contain at least 1 item",
		})
	} else if len(req.Items) > 20 {
		errors = append(errors, ValidationError{
			Field:   "items",
			Message: "order must not contain more than 20 items",
		})
	}

	// 5. Валидация каждого item
	for i, item := range req.Items {
		itemPrefix := fmt.Sprintf("items[%d]", i)

		// Валидация name
		itemName := strings.TrimSpace(item.Name)
		if len(itemName) < 1 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.name", itemPrefix),
				Message: "item name is required",
			})
		} else if len(itemName) > 50 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.name", itemPrefix),
				Message: "item name must not exceed 50 characters",
			})
		}

		// Валидация quantity
		if item.Quantity < 1 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.quantity", itemPrefix),
				Message: "item quantity must be at least 1",
			})
		} else if item.Quantity > 10 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.quantity", itemPrefix),
				Message: "item quantity must not exceed 10",
			})
		}

		// Валидация price
		if item.Price < 0.01 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.price", itemPrefix),
				Message: "item price must be at least 0.01",
			})
		} else if item.Price > 999.99 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.price", itemPrefix),
				Message: "item price must not exceed 999.99",
			})
		}
	}

	return errors
}

func convertItemsToCommand(items []OrderItemRequest) []interfaces.CreateOrderItemCommand {
	result := make([]interfaces.CreateOrderItemCommand, len(items))
	for i, item := range items {
		result[i] = interfaces.CreateOrderItemCommand{
			Name:     strings.TrimSpace(item.Name),
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}
	return result
}

func (h *OrderHandler) respondError(w http.ResponseWriter, message string, statusCode int, validationErrors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := ErrorResponse{
		Error:  message,
		Errors: validationErrors,
	}

	json.NewEncoder(w).Encode(errResp)
}
