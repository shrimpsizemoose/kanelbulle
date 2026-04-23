VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/shrimpsizemoose/kanelbulle/internal/version.Version=$(VERSION)

.PHONY: build build-bot build-server build-exporter docker-build docker-push clean

echo-version:
	@echo current version = $(VERSION)

get-pods:
	kubectl get pods -n kanelbulle

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

tell-dependabot-issues:
	@printf "STATE\tSEVERITY\tPACKAGE\tFIXED IN\tSUMMARY\n" | expand -t 12,22,56,66
	@gh api repos/shrimpsizemoose/kanelbulle/dependabot/alerts \
		--jq 'sort_by(.security_advisory.severity | {critical:0,high:1,medium:2,low:3}[.]) | .[] | [.state, .security_advisory.severity, .dependency.package.name, (.security_vulnerability.first_patched_version.identifier // "n/a"), .security_advisory.summary] | @tsv' \
		| expand -t 12,22,56,66
