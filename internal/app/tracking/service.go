package tracking

import (
	"context"
	"time"
	"wheres-my-pizza/internal/adapter/logger"
	"wheres-my-pizza/internal/domain"
	"wheres-my-pizza/internal/interfaces"
)

type Service struct {
	orderRepo  interfaces.OrderRepository
	workerRepo interfaces.WorkerRepository
	logger     logger.Logger
}

func NewService(orderRepo interfaces.OrderRepository, workerRepo interfaces.WorkerRepository, logger logger.Logger) *Service {
	return &Service{
		orderRepo:  orderRepo,
		workerRepo: workerRepo,
		logger:     logger,
	}
}

func (s *Service) GetOrderStatus(ctx context.Context, orderNumber string) (*interfaces.TrackingOrderResponse, error) {
	order, err := s.orderRepo.FindByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	resp := &interfaces.TrackingOrderResponse{
		OrderNumber:   order.Number,
		CurrentStatus: order.Status,
		UpdatedAt:     order.UpdatedAt,
		ProcessedBy:   order.ProcessedBy,
	}

	if order.Status == domain.StatusCooking {
		est := order.UpdatedAt.Add(order.GetCookingTime())
		resp.EstimatedCompletion = &est
	}

	return resp, nil
}

func (s *Service) GetOrderHistory(ctx context.Context, orderNumber string) ([]*domain.StatusLog, error) {
	order, err := s.orderRepo.FindByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}
	return s.orderRepo.GetStatusHistory(ctx, order.ID)
}

func (s *Service) GetWorkersStatus(ctx context.Context) ([]*interfaces.TrackingWorkerResponse, error) {
	workers, err := s.workerRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var resp []*interfaces.TrackingWorkerResponse
	// Допустим, таймаут оффлайна 60 секунд (2 heartbeat interval)
	timeout := 60 * time.Second

	for _, w := range workers {
		status := w.Status
		if status == domain.WorkerStatusOnline && time.Since(w.LastSeen) > timeout {
			status = domain.WorkerStatusOffline
		}

		resp = append(resp, &interfaces.TrackingWorkerResponse{
			WorkerName:      w.Name,
			Status:          status,
			OrdersProcessed: w.OrdersProcessed,
			LastSeen:        w.LastSeen,
		})
	}

	return resp, nil
}
