package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/YelzhanWeb/pizzas/internal/domain"
	"github.com/YelzhanWeb/pizzas/internal/interfaces"
)

type orderRepository struct {
	db DB
}

func NewOrderRepository(db DB) interfaces.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert order
	query := `
		INSERT INTO orders (number, customer_name, type, table_number, delivery_address, 
		                    total_amount, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err = tx.QueryRow(ctx, query,
		order.Number, order.CustomerName, order.Type, order.TableNumber, order.DeliveryAddress,
		order.TotalAmount, order.Priority, order.Status, order.CreatedAt, order.UpdatedAt,
	).Scan(&order.ID)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert order items
	for i := range order.Items {
		itemQuery := `
			INSERT INTO order_items (order_id, name, quantity, price, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		err = tx.QueryRow(ctx, itemQuery,
			order.ID, order.Items[i].Name, order.Items[i].Quantity, order.Items[i].Price, time.Now(),
		).Scan(&order.Items[i].ID)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
		order.Items[i].OrderID = order.ID
	}

	// Log initial status
	logQuery := `
		INSERT INTO order_status_log (order_id, status, changed_by, changed_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, logQuery, order.ID, order.Status, "order-service", time.Now())
	if err != nil {
		return fmt.Errorf("failed to log status: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *orderRepository) FindByNumber(ctx context.Context, number string) (*domain.Order, error) {
	query := `
		SELECT id, number, customer_name, type, table_number, delivery_address,
		       total_amount, priority, status, processed_by, created_at, updated_at, completed_at
		FROM orders
		WHERE number = $1
	`

	var order domain.Order
	err := r.db.QueryRow(ctx, query, number).Scan(
		&order.ID, &order.Number, &order.CustomerName, &order.Type, &order.TableNumber,
		&order.DeliveryAddress, &order.TotalAmount, &order.Priority, &order.Status,
		&order.ProcessedBy, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Load order items
	itemsQuery := `SELECT id, order_id, name, quantity, price FROM order_items WHERE order_id = $1`
	rows, err := r.db.Query(ctx, itemsQuery, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.Name, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

func (r *orderRepository) FindByID(ctx context.Context, id int) (*domain.Order, error) {
	query := `
		SELECT id, number, customer_name, type, table_number, delivery_address,
		       total_amount, priority, status, processed_by, created_at, updated_at, completed_at
		FROM orders
		WHERE id = $1
	`

	var order domain.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID, &order.Number, &order.CustomerName, &order.Type, &order.TableNumber,
		&order.DeliveryAddress, &order.TotalAmount, &order.Priority, &order.Status,
		&order.ProcessedBy, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) Update(ctx context.Context, order *domain.Order) error {
	query := `
		UPDATE orders
		SET status = $1, processed_by = $2, updated_at = $3, completed_at = $4
		WHERE id = $5
	`
	_, err := r.db.Exec(ctx, query,
		order.Status, order.ProcessedBy, order.UpdatedAt, order.CompletedAt, order.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}

func (r *orderRepository) GetStatusHistory(ctx context.Context, orderID int) ([]*domain.StatusLog, error) {
	query := `
		SELECT id, order_id, status, changed_by, changed_at, notes
		FROM order_status_log
		WHERE order_id = $1
		ORDER BY changed_at ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status history: %w", err)
	}
	defer rows.Close()

	var logs []*domain.StatusLog
	for rows.Next() {
		var log domain.StatusLog
		if err := rows.Scan(&log.ID, &log.OrderID, &log.Status, &log.ChangedBy, &log.ChangedAt, &log.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan status log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *orderRepository) GenerateOrderNumber(ctx context.Context) (string, error) {
	now := time.Now().UTC()
	prefix := fmt.Sprintf("ORD_%s_", now.Format("20060102"))

	query := `
		SELECT COUNT(*) FROM orders 
		WHERE number LIKE $1 AND DATE(created_at) = $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, prefix+"%", now.Format("2006-01-02")).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("failed to count orders: %w", err)
	}

	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}

func (r *orderRepository) LogStatus(ctx context.Context, orderID int, status domain.Status, changedBy string) error {
	query := `
		INSERT INTO order_status_log (order_id, status, changed_by, changed_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, orderID, status, changedBy, time.Now())
	if err != nil {
		return fmt.Errorf("failed to log status: %w", err)
	}
	return nil
}
