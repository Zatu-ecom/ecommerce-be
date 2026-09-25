.PHONY: help build build-dev run run-dev stop clean logs test migrate docker-up docker-down docker-restart arch-check

# Default target
help:
	@echo "🛍️  Zatu E-commerce Backend - Docker Commands"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  make build          - Build production Docker image"
	@echo ""
	@echo "🚀 Run Commands:"
	@echo "  make up             - Start all services"
	@echo "  make down           - Stop all services"
	@echo "  make restart        - Restart all services"
	@echo ""
	@echo "🔧 Utility Commands:"
	@echo "  make logs           - View application logs"
	@echo "  make logs-all       - View all services logs"
	@echo "  make migrate        - Run database migrations"
	@echo "  make shell          - Open shell in app container"
	@echo "  make db-shell       - Open PostgreSQL shell"
	@echo "  make redis-cli      - Open Redis CLI"
	@echo ""
	@echo "🧹 Cleanup Commands:"
	@echo "  make clean          - Stop and remove containers"
	@echo "  make clean-all      - Stop, remove containers and volumes (⚠️  deletes data)"
	@echo "  make prune          - Remove unused Docker resources"
	@echo ""
	@echo "🧪 Test Commands:"
	@echo "  make test           - Run tests in Docker"
	@echo "  make test-local     - Run tests locally"
	@echo ""
	@echo "📊 Monitoring Commands:"
	@echo "  make ps             - Show running containers"
	@echo "  make stats          - Show container resource usage"
	@echo "  make health         - Check service health"

# Build production image
build:
	@echo "🏗️  Building production image..."
	DOCKER_BUILDKIT=1 docker build -t ecommerce-backend:latest --target runtime .
	@echo "✅ Build complete!"



# Start all services (production)
up:
	@echo "🚀 Starting services (production)..."
	docker-compose up -d
	@echo "✅ Services started!"
	@echo "📝 Run 'make migrate' to set up the database"
	@echo "🌐 API available at http://localhost:8080"



# Stop all services
down:
	@echo "🛑 Stopping services..."
	docker-compose down
	@echo "✅ Services stopped!"

# Restart all services
restart:
	@echo "🔄 Restarting services..."
	docker-compose restart
	@echo "✅ Services restarted!"

# View application logs
logs:
	docker-compose logs -f app

# View all logs
logs-all:
	docker-compose logs -f

# Run database migrations
migrate:
	@echo "🗄️  Running database migrations..."
	docker-compose --profile tools run --rm migrate
	@echo "✅ Migrations complete!"

# Open shell in app container
shell:
	docker-compose exec app sh

# Open PostgreSQL shell
db-shell:
	docker-compose exec postgres psql -U postgres -d ecommerce

# Open Redis CLI
redis-cli:
	docker-compose exec redis redis-cli

# Show running containers
ps:
	docker-compose ps

# Show container stats
stats:
	docker stats --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"

# Check service health
health:
	@echo "🏥 Checking service health..."
	@docker-compose ps --format json | jq -r '.[] | "\(.Service): \(.State) - \(.Health)"'

# Clean up containers
clean:
	@echo "🧹 Cleaning up containers..."
	docker-compose down
	@echo "✅ Cleanup complete!"

# Clean up everything including volumes
clean-all:
	@echo "⚠️  This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "✅ All containers and volumes removed!"; \
	else \
		echo "❌ Cancelled"; \
	fi

# Prune unused Docker resources
prune:
	@echo "🧹 Removing unused Docker resources..."
	docker system prune -f
	@echo "✅ Prune complete!"

# Test backends are shared per `go test` package process (one Postgres + one
# Redis/MinIO/RabbitMQ each, with TRUNCATE/FLUSHALL/purge reset between
# suites) instead of one container pair per suite. Parallel packages (-p > 1)
# can still overwhelm Docker Desktop (~4 GiB): mapped ports refuse, Postgres
# readiness times out, Ryuk reaper names collide. Keep -p 1 until the shared
# setup proves green, then raise parallelism deliberately.
GOTEST_INTEGRATION_FLAGS ?= -p 1 -timeout=45m

# Run all tests under ./test/... (fast unit + shared-container integration)
test:
	@echo "🧪 Running tests locally..."
	go test $(GOTEST_INTEGRATION_FLAGS) ./test/... -v

# Run all tests with summary (failed tests shown at end)
test-all:
	@echo "🧪 Running all tests under ./test/... (use 'make test-pretty' for a formatted test report)..."
	@go test $(GOTEST_INTEGRATION_FLAGS) ./test/... -v 2>&1 | tee /tmp/test_output.txt; \

# Install gotestsum if not present and run tests with pretty format
test-pretty:
	@echo "🔍 Checking for gotestsum..."
	@which gotestsum >/dev/null || (echo "📦 Installing gotestsum..." && go install gotest.tools/gotestsum@latest)
	@echo "🧪 Running tests with gotestsum for a formatted summary..."
	@rm -f /tmp/.gotestsum_exit; \
	{ $$(go env GOPATH)/bin/gotestsum --format pkgname --junitfile /tmp/test_report.xml -- $(GOTEST_INTEGRATION_FLAGS) -v ./test/... 2>&1; echo $$? > /tmp/.gotestsum_exit; } | tee /tmp/test_output.txt; \
	GOTESTSUM_EXIT=$$(cat /tmp/.gotestsum_exit); \
	python3 scripts/summarize_junit.py /tmp/test_report.xml; \
	SUMMARY_EXIT=$$?; \
	if [ "$$GOTESTSUM_EXIT" -ne 0 ] && [ "$$SUMMARY_EXIT" -eq 0 ]; then \
		echo "⚠️  gotestsum exited ($$GOTESTSUM_EXIT) but the report shows no failures (e.g. setup aborted)"; \
		exit "$$GOTESTSUM_EXIT"; \
	fi; \
	exit "$$SUMMARY_EXIT"

# Same as test-pretty but TEST_KV_DUAL=1 (two Redis per suite). Use on CI
# or a host that can hold volatile + durable isolation tests.
test-pretty-dual:
	@echo "🧪 Running tests with TEST_KV_DUAL=1 (two Redis processes per suite)..."
	TEST_KV_DUAL=1 $(MAKE) test-pretty

# Run tests with JSON output for CI/CD
test-json:
	@echo "🧪 Running tests with JSON output..."
	go test $(GOTEST_INTEGRATION_FLAGS) ./test/... -json 2>&1 | tee test-results.json

# Test backends for TEST_USE_EXTERNAL=1 (single shared Postgres + Redis for
# the whole run, fastest local loop; tests still reset state per suite).
test-up:
	@echo "🚀 Starting test backends..."
	docker compose -f docker-compose.test.yml up -d
	@echo "✅ Test backends up (postgres :5433, cache :6382, durable :6383)"
	@echo "📝 Run 'TEST_USE_EXTERNAL=1 make test' to use them"

test-down:
	@echo "🛑 Stopping test backends..."
	docker compose -f docker-compose.test.yml down
	@echo "✅ Test backends stopped!"

test-external:
	@echo "🧪 Running tests against external backends (TEST_USE_EXTERNAL=1)..."
	TEST_USE_EXTERNAL=1 go test $(GOTEST_INTEGRATION_FLAGS) ./test/... -v

# Architecture guard: common/model must remain pure (no domain imports).
# Money/currency standardization — violates the modular-monolith dependency rule.
arch-check:
	@echo "🏗️  Checking common/model has no domain imports..."
	@if go list -f '{{.ImportPath}} {{.Imports}}' ./common/model | grep -E 'ecommerce-be/(user|product|order|promotion|report|payment|inventory|notification|file|fulfillment|subscription)'; then \
		echo "❌ common/model must NOT import domain modules"; \
		exit 1; \
	else \
		echo "✅ common/model is pure (no domain imports)"; \
	fi

# Re-run only failed tests from the last test-all run
test-failed:
	@echo "🔄 Re-running failed tests..."
	@if [ ! -f /tmp/test_output.txt ]; then \
		echo "❌ No previous test run found. Run 'make test-all' first."; \
		exit 1; \
	fi; \
	FAILED_TESTS=$$(grep -Fe '--- FAIL:' /tmp/test_output.txt | grep -v '^===' | grep '/' | sed 's/^    //' | sed 's/--- FAIL: //' | sed 's/ (.*//' | tr '\n' '|' | sed 's/|$$//'); \
	if [ -z "$$FAILED_TESTS" ]; then \
		echo "✅ No failed tests to re-run!"; \
	else \
		echo "Running: $$FAILED_TESTS"; \
		go test ./test/... -v -run "$$FAILED_TESTS" 2>&1 | tee /tmp/test_output.txt; \
		echo ""; \
		echo "=========================================="; \
		echo "           📊 RE-RUN SUMMARY"; \
		echo "=========================================="; \
		LEAF_FAILED=$$(grep -Fe '--- FAIL:' /tmp/test_output.txt 2>/dev/null | grep '/' | wc -l || true); \
		LEAF_FAILED=$${LEAF_FAILED:-0}; \
		ALL_FAILED=$$(grep -cFe '--- FAIL:' /tmp/test_output.txt 2>/dev/null || true); \
		ALL_FAILED=$${ALL_FAILED:-0}; \
		if [ "$$LEAF_FAILED" -gt 0 ]; then FAILED=$$LEAF_FAILED; else FAILED=$$ALL_FAILED; fi; \
		LEAF_PASSED=$$(grep -Fe '--- PASS:' /tmp/test_output.txt 2>/dev/null | grep '/' | wc -l || true); \
		LEAF_PASSED=$${LEAF_PASSED:-0}; \
		ALL_PASSED=$$(grep -cFe '--- PASS:' /tmp/test_output.txt 2>/dev/null || true); \
		ALL_PASSED=$${ALL_PASSED:-0}; \
		if [ "$$LEAF_PASSED" -gt 0 ]; then PASSED=$$LEAF_PASSED; else PASSED=$$ALL_PASSED; fi; \
		echo "✅ Passed: $$PASSED"; \
		echo "❌ Failed: $$FAILED"; \
		if [ "$$FAILED" -gt 0 ]; then \
			echo ""; \
			echo "=========================================="; \
			echo "           ❌ STILL FAILING"; \
			echo "=========================================="; \
			grep -Fe '--- FAIL:' /tmp/test_output.txt | grep '/' | sed 's/^    //' | sort -u; \
		fi; \
	fi

# Quick start (build + up + migrate)
quickstart: build up migrate
	@echo "🎉 Quick start complete!"
	@echo "🌐 API available at http://localhost:8080"
