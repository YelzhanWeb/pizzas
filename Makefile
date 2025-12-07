.PHONY: build
build:
	go build -o restaurant-system ./cmd/main.go

test:
	go test -v -cover .

format:
	gofumpt -l -w .

run-order:
	./bin/restaurant-system --mode=order-service --port=3000

run-worker:
	./bin/restaurant-system --mode=kitchen-worker --worker-name=chef_anna --prefetch=1

run-tracking:
	./bin/restaurant-system --mode=tracking-service --port=3002

run-notification:
	./bin/restaurant-system --mode=notification-subscriber

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down