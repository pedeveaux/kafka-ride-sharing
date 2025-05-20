BIN_DIR := ./bin

build-producer:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BIN_DIR)/producer ./producer

build-consumer:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BIN_DIR)/consumer ./consumer

build: build-producer build-consumer

compose-build:
	docker compose build

up: build compose-build
	docker compose up -d

producer:
	docker compose up -d producer

consumer:
	docker compose up -d consumer

down:
	docker compose down

logs:
	docker compose logs -f