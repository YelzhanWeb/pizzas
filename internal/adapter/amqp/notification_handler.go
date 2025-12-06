package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"wheres-my-pizza/internal/adapter/logger"
	"wheres-my-pizza/internal/interfaces"
)

type NotificationHandler struct {
	logger logger.Logger
}

func NewNotificationHandler(logger logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		logger: logger,
	}
}

func (h *NotificationHandler) HandleNotification(ctx context.Context, body []byte) error {
	var msg interfaces.StatusUpdateMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		h.logger.Error("message_parse_failed", "Failed to parse notification", "", nil, err)
		return err
	}

	h.logger.Debug("notification_received", fmt.Sprintf("Received status update for order %s", msg.OrderNumber),
		msg.OrderNumber, map[string]interface{}{
			"order_number": msg.OrderNumber,
			"new_status":   msg.NewStatus,
		})

	// Print to console
	fmt.Printf("Notification for order %s: Status changed from '%s' to '%s' by %s\n",
		msg.OrderNumber, msg.OldStatus, msg.NewStatus, msg.ChangedBy)

	return nil
}
