package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/adapter/logger"

	"github.com/YelzhanWeb/pizzas/internal/interfaces"
)

type TrackingHandler struct {
	service interfaces.TrackingService
	logger  logger.Logger
}

func NewTrackingHandler(service interfaces.TrackingService, logger logger.Logger) *TrackingHandler {
	return &TrackingHandler{
		service: service,
		logger:  logger,
	}
}

func (h *TrackingHandler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	requestID := time.Now().UnixNano()
	h.logger.Debug("request_received", "Request received", string(rune(requestID)), map[string]interface{}{
		"path": r.URL.Path,
	})

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	orderNumber := parts[1]

	if len(parts) == 3 && parts[2] == "status" {
		h.getOrderStatus(w, r, orderNumber)
	} else if len(parts) == 3 && parts[2] == "history" {
		h.getOrderHistory(w, r, orderNumber)
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (h *TrackingHandler) getOrderStatus(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := h.service.GetOrderStatus(r.Context(), orderNumber)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"order_number":         result.OrderNumber,
		"current_status":       result.CurrentStatus,
		"updated_at":           result.UpdatedAt,
		"estimated_completion": result.EstimatedCompletion,
		"processed_by":         result.ProcessedBy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *TrackingHandler) getOrderHistory(w http.ResponseWriter, r *http.Request, orderNumber string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	history, err := h.service.GetOrderHistory(r.Context(), orderNumber)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	resp := make([]map[string]interface{}, len(history))
	for i, log := range history {
		resp[i] = map[string]interface{}{
			"status":     log.Status,
			"timestamp":  log.ChangedAt,
			"changed_by": log.ChangedBy,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *TrackingHandler) GetWorkersStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Debug("request_received", "Workers status requested", "", nil)

	workers, err := h.service.GetWorkersStatus(r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]interface{}, len(workers))
	for i, worker := range workers {
		resp[i] = map[string]interface{}{
			"worker_name":      worker.WorkerName,
			"status":           worker.Status,
			"orders_processed": worker.OrdersProcessed,
			"last_seen":        worker.LastSeen,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
