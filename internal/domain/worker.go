package domain

import (
	"errors"
	"time"
)

// Worker represents a kitchen worker entity
type Worker struct {
	ID              int
	Name            string
	Type            string
	Status          WorkerStatus
	LastSeen        time.Time
	OrdersProcessed int
	CreatedAt       time.Time
}

type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"
	WorkerStatusOffline WorkerStatus = "offline"
)

// NewWorker creates a new worker
func NewWorker(name, workerType string) (*Worker, error) {
	if name == "" {
		return nil, errors.New("worker name is required")
	}

	return &Worker{
		Name:      name,
		Type:      workerType,
		Status:    WorkerStatusOnline,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}, nil
}

// UpdateHeartbeat updates the worker's last seen timestamp
func (w *Worker) UpdateHeartbeat() {
	w.LastSeen = time.Now()
	w.Status = WorkerStatusOnline
}

// SetOffline marks the worker as offline
func (w *Worker) SetOffline() {
	w.Status = WorkerStatusOffline
}

// IncrementOrdersProcessed increments the orders processed count
func (w *Worker) IncrementOrdersProcessed() {
	w.OrdersProcessed++
}

// IsOnline checks if the worker is considered online based on last heartbeat
func (w *Worker) IsOnline(heartbeatTimeout time.Duration) bool {
	if w.Status == WorkerStatusOffline {
		return false
	}
	return time.Since(w.LastSeen) <= heartbeatTimeout
}
