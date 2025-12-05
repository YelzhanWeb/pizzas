.PHONY: build test run-order run-worker run-tracking run-notification clean docker-up docker-down format lint

# Build the application
build:
 go build -o bin/restaurant-system .

# Run tests
test:
 go test -v -cover ./...

# Format code with gofumpt
format:
 gofumpt -l -w .

# Lint code
lint:
 gofumpt -l .
 go vet ./...

# Run order service
run-order:
 ./bin/restaurant-system --mode=order-service --port=3000

# Run kitchen worker
run-worker:
 ./bin/restaurant-system --mode=kitchen-worker --worker-name=chef_anna --prefetch=1

# Run tracking service
run-tracking:
 ./bin/restaurant-system --mode=tracking-service --port=3002

# Run notification subscriber
run-notification:
 ./bin/restaurant-system --mode=notification-subscriber

# Clean build artifacts
clean:
 rm -rf bin/

# Start Docker services
docker-up:
 docker-compose up -d

# Stop Docker services
docker-down:
 docker-compose down

# Initialize database
init-db:
 psql -h localhost -U restaurant_user -d restaurant_db -f migrations/001_schema.sql