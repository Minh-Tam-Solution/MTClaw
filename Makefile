VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -s -w -X github.com/nextlevelbuilder/goclaw/cmd.Version=$(VERSION)
BINARY   = goclaw

.PHONY: build run clean version up down logs migrate-up migrate-down souls-validate test

# Build & Run
build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

version:
	@echo $(VERSION)

# Test
test:
	go test ./... -v -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate-up: build
	./$(BINARY) migrate up

migrate-down: build
	./$(BINARY) migrate down

# SOUL Validation
souls-validate:
	@echo "Validating SOUL files..."
	@for f in docs/08-collaborate/souls/SOUL-*.md; do \
		if ! head -1 "$$f" | grep -q '^---$$'; then \
			echo "WARN: $$f missing YAML frontmatter"; \
		fi; \
	done
	@echo "Found $$(ls docs/08-collaborate/souls/SOUL-*.md | wc -l) SOUL files"

# Docker Compose
COMPOSE = docker compose -f docker-compose.yml -f docker-compose.managed.yml -f docker-compose.selfservice.yml

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f goclaw
