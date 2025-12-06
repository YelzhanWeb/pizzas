package amqp

import (
	"context"
	"encoding/json"
	"wheres-my-pizza/internal/adapter/logger"
	"wheres-my-pizza/internal/interfaces"
)

type OrderHandler struct {
	service interfaces.KitchenService
	logger  logger.Logger
}

func NewOrderHandler(service interfaces.KitchenService, logger logger.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger,
	}
}

func (h *OrderHandler) HandleOrder(ctx context.Context, body []byte) error {
	var msg interfaces.OrderMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		h.logger.Error("message_parse_failed", "Failed to parse order message", "", nil, err)
		return err
	}

	return h.service.ProcessOrder(ctx, msg)
}
