-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    number TEXT UNIQUE NOT NULL,
    customer_name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (
        type IN (
            'dine_in',
            'takeout',
            'delivery'
        )
    ),
    table_number INTEGER,
    delivery_address TEXT,
    total_amount DECIMAL(10, 2) NOT NULL,
    priority INTEGER DEFAULT 1,
    status TEXT DEFAULT 'received',
    processed_by TEXT,
    completed_at TIMESTAMPTZ
);

-- Create order items table
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    order_id INTEGER REFERENCES orders (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    price DECIMAL(8, 2) NOT NULL
);

-- Create order status log table
CREATE TABLE IF NOT EXISTS order_status_log (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    order_id INTEGER REFERENCES orders (id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    changed_by TEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    notes TEXT
);