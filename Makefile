VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -s -w -X github.com/Minh-Tam-Solution/MTClaw/cmd.Version=$(VERSION)
BINARY   = mtclaw

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

# SOUL Validation (CTO-3: char budget check)
# Note: Source files in docs/ are REFERENCE docs (include playbooks, comms patterns, etc.)
# System prompt injection uses condensed content from migrations/agent_context_files.
# Validator: FAIL if missing frontmatter, WARN if >2500 chars (reference vs deployment distinction).
souls-validate:
	@echo "Validating SOUL files..."
	@failed=0; warns=0; \
	for f in docs/08-collaborate/souls/SOUL-*.md; do \
		chars=$$(wc -c < "$$f"); \
		if ! head -1 "$$f" | grep -q '^---$$'; then \
			echo "FAIL: $$f — missing YAML frontmatter"; \
			failed=1; \
		elif [ "$$chars" -gt 2500 ]; then \
			echo "WARN: $$f ($$chars chars, budget 2500)"; \
			warns=$$((warns + 1)); \
		else \
			echo "  OK: $$f ($$chars chars)"; \
		fi; \
	done; \
	total=$$(ls docs/08-collaborate/souls/SOUL-*.md | wc -l); \
	echo "Found $$total SOUL files ($$warns over budget)"; \
	if [ "$$failed" -eq 1 ]; then echo "FAIL: SOUL validation failed (frontmatter)"; exit 1; fi
	@echo "SOUL validation passed."

# Docker Compose
COMPOSE = docker compose -f docker-compose.yml -f docker-compose.managed.yml -f docker-compose.mts.yml -f docker-compose.selfservice.yml

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f mtclaw
