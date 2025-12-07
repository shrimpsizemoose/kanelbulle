VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/shrimpsizemoose/kanelbulle/internal/bot.Version=$(VERSION)

.PHONY: build build-bot build-server build-exporter docker-build docker-push clean

echo-version:
	@echo current version = $(VERSION)

build: build-bot build-server build-exporter

build-bot:
	go build -ldflags "$(LDFLAGS)" -o bin/bot ./cmd/bot

build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

build-exporter:
	go build -ldflags "$(LDFLAGS)" -o bin/exporter ./cmd/exporter

docker-build:
	VERSION=$(VERSION) docker compose build bot server

docker-push:
	VERSION=$(VERSION) docker compose push bot server

clean:
	rm -rf bin/
