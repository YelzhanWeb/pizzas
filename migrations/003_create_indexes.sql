-- Create indexes
CREATE INDEX IF NOT EXISTS idx_orders_number ON orders (number);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);

CREATE INDEX IF NOT EXISTS idx_order_status_log_order_id ON order_status_log (order_id);

CREATE INDEX IF NOT EXISTS idx_workers_name ON workers (name);

CREATE INDEX IF NOT EXISTS idx_workers_status ON workers (status);