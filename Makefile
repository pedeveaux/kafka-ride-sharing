BIN_DIR := ./bin

build-producer:
	go build -tags dynamic -o $(BIN_DIR)/producer ./producer

build-consumer:
	go build -tags dynamic -o $(BIN_DIR)/consumer ./consumer

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

clean:
	docker compose down -v